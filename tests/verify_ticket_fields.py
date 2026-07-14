import requests
import json
import time

BASE_URL = "http://localhost:8090/api/v1"

def test_create_and_update_ticket():
    print("Step 1: Create Ticket with Type")
    payload = {
        "title": "Test Ticket Fields",
        "description": "Verifying type and resolution fields",
        "priority": "medium",
        "type": "incident",
        "category": "Test",
        "requester_id": 1,
        "tenant_id": 1
    }

    # Assuming authentication is handled or disabled for dev, or we need a token.
    # For now, let's try without token if dev mode allows, or simulate headers.
    headers = {
        "Content-Type": "application/json",
        "X-Tenant-ID": "1",
        "X-User-ID": "1" # Assuming middleware reads this in dev mode or test env
    }

    try:
        response = requests.post(f"{BASE_URL}/tickets", json=payload, headers=headers)
        if response.status_code != 201 and response.status_code != 200:
            print(f"Failed to create ticket: {response.status_code} {response.text}")
            return

        ticket = response.json().get("data", {})
        ticket_id = ticket.get("id")
        print(f"Ticket Created: ID={ticket_id}, Type={ticket.get('type')}")

        if ticket.get("type") != "incident":
            print("❌ Type field mismatch on creation!")
        else:
            print("✅ Type field verified on creation.")

        print("\nStep 2: Update Ticket Resolution")
        update_payload = {
            "resolution": "Fixed by restarting service",
            "status": "resolved",
            "user_id": 1
        }

        response = requests.put(f"{BASE_URL}/tickets/{ticket_id}", json=update_payload, headers=headers)
        if response.status_code != 200:
            print(f"Failed to update ticket: {response.status_code} {response.text}")
            return

        updated_ticket = response.json().get("data", {})
        print(f"Ticket Updated: Resolution={updated_ticket.get('resolution')}")

        if updated_ticket.get("resolution") != "Fixed by restarting service":
             print("❌ Resolution field mismatch on update!")
        else:
             print("✅ Resolution field verified on update.")

        # Verify by fetching again
        print("\nStep 3: Fetch Ticket to confirm persistence")
        response = requests.get(f"{BASE_URL}/tickets/{ticket_id}", headers=headers)
        fetched_ticket = response.json().get("data", {})

        if fetched_ticket.get("type") == "incident" and fetched_ticket.get("resolution") == "Fixed by restarting service":
            print("✅ All fields persisted correctly.")
        else:
            print(f"❌ Persistence check failed: Type={fetched_ticket.get('type')}, Resolution={fetched_ticket.get('resolution')}")

    except Exception as e:
        print(f"Test failed with exception: {e}")

if __name__ == "__main__":
    # Wait for server to start
    time.sleep(2)
    test_create_and_update_ticket()
