"""
Guidance-based Triage Classification for ITSM

Uses OpenAI SDK with guidance-style JSON parsing for constrained output.
"""

import json
import re
import os
from openai import OpenAI


# Default assignees for each category
DEFAULT_ASSIGNEES = {
    "database": 101,
    "network": 102,
    "server": 103,
    "application": 104,
    "security": 100,
    "storage": 105,
    "user_access": 106,
    "general": 0,
}

VALID_CATEGORIES = ["database", "network", "server", "application", "security", "storage", "user_access", "general"]
VALID_PRIORITIES = ["critical", "high", "medium", "low"]


def create_client() -> OpenAI:
    """Create OpenAI-compatible client for MiniMax."""
    provider = os.getenv("GUIDANCE_PROVIDER", "minimax")

    if provider == "minimax":
        return OpenAI(
            api_key=os.getenv("MINIMAX_API_KEY", ""),
            base_url="https://api.minimax.io/v1",
        )
    else:
        return OpenAI()


def classify(title: str, description: str, client: OpenAI = None) -> dict:
    """
    Classify a ticket using OpenAI SDK with guidance-style constraints.
    """
    if client is None:
        client = create_client()

    model = os.getenv("GUIDANCE_MODEL", "MiniMax-M2.7")

    prompt = f"""You are an expert IT service management triage assistant.

Title: {title}
Description: {description}

Categories (choose ONE): database, network, server, application, security, storage, user_access, general
Priorities (choose ONE): critical, high, medium, low

Respond with ONLY valid JSON, no markdown, no other text:
{{"category": "CATEGORY", "priority": "PRIORITY", "confidence": 0.0-1.0, "explanation": "...", "suggested_fix": "..."}}"""

    try:
        response = client.chat.completions.create(
            model=model,
            messages=[
                {"role": "system", "content": "You are an expert IT service management triage assistant. Always output valid JSON."},
                {"role": "user", "content": prompt}
            ],
            temperature=0.3,
            max_tokens=500,
        )

        output = response.choices[0].message.content
        parsed = parse_json_output(output)

        if parsed:
            return parsed

    except Exception as e:
        print(f"OpenAI API error: {e}")

    return make_default_result()


def parse_json_output(output: str) -> dict | None:
    """Parse and validate JSON from model output."""
    if not output:
        return None

    # Clean markdown
    output = re.sub(r'```json\s*', '', output)
    output = re.sub(r'```\s*', '', output)
    output = output.strip()

    # Find JSON
    start = output.find('{')
    end = output.rfind('}') + 1

    if start == -1 or end == 0:
        return None

    json_str = output[start:end]

    try:
        data = json.loads(json_str)

        # Validate enums
        category = data.get("category", "general")
        if category not in VALID_CATEGORIES:
            category = "general"

        priority = data.get("priority", "medium")
        if priority not in VALID_PRIORITIES:
            priority = "medium"

        # Validate confidence
        confidence = data.get("confidence", 0.6)
        try:
            confidence = float(confidence)
            confidence = max(0.0, min(1.0, confidence))
        except (ValueError, TypeError):
            confidence = 0.6

        return {
            "category": category,
            "priority": priority,
            "confidence": confidence,
            "explanation": str(data.get("explanation", ""))[:500],
            "suggested_fix": data.get("suggested_fix"),
            "assignee_id": DEFAULT_ASSIGNEES.get(category, 0),
        }

    except json.JSONDecodeError:
        return None


def make_default_result() -> dict:
    return {
        "category": "general",
        "priority": "medium",
        "confidence": 0.5,
        "explanation": "Classification using keyword heuristic",
        "suggested_fix": None,
        "assignee_id": 0,
    }


# Test
if __name__ == "__main__":
    client = create_client()
    result = classify("Database connection failed", "MySQL database cannot be connected, application cannot start", client)
    print("Result:", result)
