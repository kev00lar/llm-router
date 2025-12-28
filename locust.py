import time
from locust import HttpUser, task, between

class LLMRouterUser(HttpUser):
    # Simulate a user thinking for 1-2 seconds between requests
    wait_time = between(1, 2)

    @task(5)  # Weight 5: Most users hit the chat endpoint
    def test_chat_proxy(self):
        payload = {
            "model": "gpt-4",
            "messages": [
                {"role": "user", "content": "Explain quantum computing in one sentence."}
            ]
        }
        with self.client.post("/v1/chat/completions", json=payload, catch_response=True) as response:
            if response.status_code == 200:
                response.success()
            elif response.status_code == 429:
                # We expect 429s if we exceed the Redis global limit
                response.failure("Global Rate Limit Exceeded")
            else:
                response.failure(f"Unexpected status: {response.status_code}")

    @task(1)  # Weight 1: Occasionally check the admin stats
    def test_admin_stats(self):
        self.client.get("/admin/stats")

# To run this:
# 1. pip install locust
# 2. locust -f locustfile.py
# 3. Open http://localhost:8089