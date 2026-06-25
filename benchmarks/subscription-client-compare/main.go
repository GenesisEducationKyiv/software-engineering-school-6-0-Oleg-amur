package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	subscriptionsv1 "github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/shared/contracts/gen/subscriptions/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type activeSubscription struct {
	Email            string
	UnsubscribeToken string
}

type httpActiveSubscriptionsResponse struct {
	Subscriptions []struct {
		Email            string `json:"email"`
		UnsubscribeToken string `json:"unsubscribe_token"`
	} `json:"subscriptions"`
}

type runner func(context.Context) error

type result struct {
	mode        string
	total       int
	ok          int64
	failed      int64
	elapsed     time.Duration
	latencies   []time.Duration
	firstErrors []string
}

func main() {
	var (
		mode           = flag.String("mode", "all", "Benchmark mode: http, http-raw, grpc, or all.")
		repositoryID   = flag.Int64("repository-id", 1, "Repository id passed to both transports.")
		total          = flag.Int("total", 10000, "Total request attempts per mode.")
		concurrency    = flag.Int("concurrency", 50, "Number of concurrent workers.")
		httpURL        = flag.String("http-url", "", "Full HTTP URL. Defaults to subscription-service localhost URL with repository-id.")
		grpcAddress    = flag.String("grpc-address", "127.0.0.1:50051", "gRPC server address.")
		requestTimeout = flag.Duration("request-timeout", 10*time.Second, "Timeout for one request.")
	)
	flag.Parse()

	if *total <= 0 {
		fatalf("total must be positive")
	}
	if *concurrency <= 0 {
		fatalf("concurrency must be positive")
	}

	modes := expandModes(*mode)
	for i, currentMode := range modes {
		if i > 0 {
			fmt.Println()
		}
		run, cleanup, err := buildRunner(currentMode, *repositoryID, *httpURL, *grpcAddress, *requestTimeout)
		if err != nil {
			fatalf("%v", err)
		}

		res := runBenchmark(currentMode, *total, *concurrency, run)
		printResult(res)

		if cleanup != nil {
			if err := cleanup(); err != nil {
				fmt.Fprintf(os.Stderr, "cleanup %s: %v\n", currentMode, err)
			}
		}
	}
}

func expandModes(mode string) []string {
	switch mode {
	case "all":
		return []string{"http-raw", "http", "grpc"}
	case "http", "http-raw", "grpc":
		return []string{mode}
	default:
		fatalf("unsupported mode %q; use http, http-raw, grpc, or all", mode)
		return nil
	}
}

func buildRunner(
	mode string,
	repositoryID int64,
	httpURL string,
	grpcAddress string,
	requestTimeout time.Duration,
) (runner, func() error, error) {
	switch mode {
	case "http", "http-raw":
		if httpURL == "" {
			httpURL = "http://127.0.0.1:8080/internal/v1/subscriptions?repository_id=" +
				strconv.FormatInt(repositoryID, 10)
		}
		client := &http.Client{Timeout: requestTimeout}
		return func(ctx context.Context) error {
			reqCtx, cancel := context.WithTimeout(ctx, requestTimeout)
			defer cancel()

			req, err := http.NewRequestWithContext(reqCtx, http.MethodGet, httpURL, nil)
			if err != nil {
				return err
			}

			resp, err := client.Do(req)
			if err != nil {
				return err
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				_, _ = io.Copy(io.Discard, resp.Body)
				return fmt.Errorf("HTTP status %d", resp.StatusCode)
			}

			if mode == "http-raw" {
				_, err = io.Copy(io.Discard, resp.Body)
				return err
			}

			var decoded httpActiveSubscriptionsResponse
			if err := json.NewDecoder(resp.Body).Decode(&decoded); err != nil {
				return err
			}

			// Mirror the release-tracker HTTP adapter: decode JSON and map DTOs to domain-shaped values.
			subscriptions := make([]activeSubscription, 0, len(decoded.Subscriptions))
			for _, subscription := range decoded.Subscriptions {
				subscriptions = append(subscriptions, activeSubscription{
					Email:            subscription.Email,
					UnsubscribeToken: subscription.UnsubscribeToken,
				})
			}
			return nil
		}, nil, nil

	case "grpc":
		conn, err := grpc.NewClient(
			grpcAddress,
			grpc.WithTransportCredentials(insecure.NewCredentials()),
		)
		if err != nil {
			return nil, nil, fmt.Errorf("create gRPC client: %w", err)
		}
		client := subscriptionsv1.NewSubscriptionServiceClient(conn)

		return func(ctx context.Context) error {
			reqCtx, cancel := context.WithTimeout(ctx, requestTimeout)
			defer cancel()

			response, err := client.ListActiveSubscriptionsByRepository(
				reqCtx,
				&subscriptionsv1.ListActiveSubscriptionsByRepositoryRequest{RepositoryId: repositoryID},
			)
			if err != nil {
				return err
			}

			// Mirror the release-tracker gRPC adapter: read generated protobuf values and map them.
			subscriptions := make([]activeSubscription, 0, len(response.GetSubscriptions()))
			for _, subscription := range response.GetSubscriptions() {
				subscriptions = append(subscriptions, activeSubscription{
					Email:            subscription.GetEmail(),
					UnsubscribeToken: subscription.GetUnsubscribeToken(),
				})
			}
			return nil
		}, conn.Close, nil
	default:
		return nil, nil, fmt.Errorf("unsupported mode %q", mode)
	}
}

func runBenchmark(mode string, total int, concurrency int, run runner) result {
	var attempts atomic.Int64
	var ok atomic.Int64
	var failed atomic.Int64

	latencies := make([]time.Duration, 0, total)
	firstErrors := make([]string, 0, 5)
	var mu sync.Mutex

	start := time.Now()
	var wg sync.WaitGroup
	for range concurrency {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				attempt := int(attempts.Add(1))
				if attempt > total {
					return
				}

				requestStart := time.Now()
				err := run(context.Background())
				latency := time.Since(requestStart)
				if err != nil {
					failed.Add(1)
					mu.Lock()
					if len(firstErrors) < cap(firstErrors) {
						firstErrors = append(firstErrors, err.Error())
					}
					mu.Unlock()
					continue
				}

				ok.Add(1)
				mu.Lock()
				latencies = append(latencies, latency)
				mu.Unlock()
			}
		}()
	}
	wg.Wait()
	elapsed := time.Since(start)

	sort.Slice(latencies, func(i, j int) bool {
		return latencies[i] < latencies[j]
	})

	return result{
		mode:        mode,
		total:       total,
		ok:          ok.Load(),
		failed:      failed.Load(),
		elapsed:     elapsed,
		latencies:   latencies,
		firstErrors: firstErrors,
	}
}

func printResult(res result) {
	fmt.Printf("mode=%s total=%d ok=%d failed=%d elapsed=%.2fs rps=%.2f\n",
		res.mode,
		res.total,
		res.ok,
		res.failed,
		res.elapsed.Seconds(),
		float64(res.ok)/res.elapsed.Seconds(),
	)

	if len(res.latencies) > 0 {
		fmt.Printf(
			"latency avg=%s p50=%s p95=%s p99=%s max=%s\n",
			formatDuration(avg(res.latencies)),
			formatDuration(percentile(res.latencies, 50)),
			formatDuration(percentile(res.latencies, 95)),
			formatDuration(percentile(res.latencies, 99)),
			formatDuration(res.latencies[len(res.latencies)-1]),
		)
	}

	if len(res.firstErrors) > 0 {
		fmt.Printf("first_errors=%s\n", strings.Join(res.firstErrors, " | "))
	}
}

func avg(values []time.Duration) time.Duration {
	var total time.Duration
	for _, value := range values {
		total += value
	}
	return total / time.Duration(len(values))
}

func percentile(values []time.Duration, percentile int) time.Duration {
	if len(values) == 0 {
		return 0
	}
	if percentile <= 0 {
		return values[0]
	}
	if percentile >= 100 {
		return values[len(values)-1]
	}
	index := (len(values)*percentile + 99) / 100
	if index <= 0 {
		index = 1
	}
	return values[index-1]
}

func formatDuration(value time.Duration) string {
	return fmt.Sprintf("%.2fms", float64(value.Microseconds())/1000)
}

func fatalf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}
