const { expect, test } = require("@playwright/test");

test("subscribes, confirms by email, and shows subscription in the list", async ({ page, request }) => {
	await clearMailbox(request);

	const email = `e2e-${Date.now()}@example.com`;
	const repo = `owner/repo-${Date.now()}`;

	await page.goto("/");
	await page.locator("#email").fill(email);
	await page.locator("#repo").fill(repo);
	await page.getByRole("button", { name: "Subscribe" }).click();

	await expect(page.getByTestId("status")).toContainText("Subscription request created");

	await page.getByRole("button", { name: "Load" }).click();

	await expect(page.getByTestId("empty-subscriptions")).toContainText("No active subscriptions found");

	const token = await waitForConfirmationToken(request);
	const confirmResponse = await page.goto(`/api/v1/confirm/${token}`);
	expect(confirmResponse.status()).toBe(200);

	await page.goto("/");
	await page.locator("#lookup-email").fill(email);
	await page.getByRole("button", { name: "Load" }).click();

	await expect(page.getByTestId("subscriptions")).toContainText(repo);
	await expect(page.getByTestId("subscriptions")).toContainText("v1.0.0");
});

test("shows validation error for invalid repository", async ({ page }) => {
	await page.goto("/");
	await page.locator("#email").fill("user@example.com");
	await page.locator("#repo").fill("invalid-repo");
	await page.getByRole("button", { name: "Subscribe" }).click();

	await expect(page.getByTestId("status")).toContainText("invalid repository format");
});

async function clearMailbox(request) {
	const response = await request.delete(`${mailpitURL()}/api/v1/messages`, {
		data: {},
	});

	if (!response.ok()) {
		throw new Error(`failed to clear Mailpit mailbox: ${response.status()}`);
	}
}

async function waitForConfirmationToken(request) {
	let token = "";

	await expect
		.poll(
			async () => {
				const response = await request.get(`${mailpitURL()}/api/v1/message/latest/raw`);
				if (!response.ok()) {
					return "";
				}

				const rawMessage = await response.text();
				const match = rawMessage.match(/\/api\/v1\/confirm\/([0-9a-fA-F-]+)/);
				token = match ? match[1] : "";
				return token;
			},
			{
				message: "confirmation email should contain confirmation token",
				timeout: 5000,
			},
		)
		.not.toBe("");

	return token;
}

function mailpitURL() {
	return process.env.MAILPIT_URL || "http://localhost:8025";
}
