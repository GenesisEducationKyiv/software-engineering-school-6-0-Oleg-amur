const subscribeForm = document.querySelector("#subscribe-form");
const lookupForm = document.querySelector("#lookup-form");
const emailInput = document.querySelector("#email");
const repoInput = document.querySelector("#repo");
const lookupEmailInput = document.querySelector("#lookup-email");
const statusBox = document.querySelector("#status");
const subscriptionsList = document.querySelector("#subscriptions");
const emptySubscriptions = document.querySelector("#empty-subscriptions");

subscribeForm.addEventListener("submit", async (event) => {
	event.preventDefault();
	setStatus("", "");

	const button = subscribeForm.querySelector("button");
	button.disabled = true;

	try {
		const response = await fetch("/api/v1/subscribe", {
			method: "POST",
			headers: {
				"Content-Type": "application/json",
			},
			body: JSON.stringify({
				email: emailInput.value.trim(),
				repo: repoInput.value.trim(),
			}),
		});

		if (!response.ok) {
			throw new Error(await readError(response));
		}

		lookupEmailInput.value = emailInput.value.trim();
		setStatus("Subscription request created. Check email to confirm it.", "success");
	} catch (error) {
		setStatus(error.message, "error");
	} finally {
		button.disabled = false;
	}
});

lookupForm.addEventListener("submit", async (event) => {
	event.preventDefault();
	setStatus("", "");
	emptySubscriptions.textContent = "";
	subscriptionsList.replaceChildren();

	const button = lookupForm.querySelector("button");
	button.disabled = true;

	try {
		const email = encodeURIComponent(lookupEmailInput.value.trim());
		const response = await fetch(`/api/v1/subscriptions?email=${email}`);

		if (!response.ok) {
			throw new Error(await readError(response));
		}

		renderSubscriptions((await response.json()) || []);
	} catch (error) {
		setStatus(error.message, "error");
	} finally {
		button.disabled = false;
	}
});

function renderSubscriptions(subscriptions) {
	subscriptionsList.replaceChildren();

	if (!subscriptions.length) {
		emptySubscriptions.textContent = "No active subscriptions found.";
		return;
	}

	emptySubscriptions.textContent = "";
	for (const subscription of subscriptions) {
		const item = document.createElement("li");
		const repo = document.createElement("span");
		const meta = document.createElement("span");

		repo.className = "repo";
		repo.textContent = subscription.repo;

		meta.className = "meta";
		meta.textContent = `Last seen tag: ${subscription.last_seen_tag || "none"}`;

		item.append(repo, meta);
		subscriptionsList.append(item);
	}
}

async function readError(response) {
	const text = await response.text();
	if (!text) {
		return `Request failed with status ${response.status}`;
	}

	try {
		const error = JSON.parse(text);
		return error.message || text;
	} catch {
		return text;
	}
}

function setStatus(message, kind) {
	statusBox.textContent = message;
	statusBox.dataset.kind = kind;
}
