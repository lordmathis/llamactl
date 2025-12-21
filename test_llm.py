#!/usr/bin/env python3
"""
Simple Python script to interact with local LLM server's OpenAI-compatible API
"""

import requests

# Local LLM server configuration
LLM_SERVER_URL = "http://localhost:8080/v1/chat/completions"
MODEL_NAME = "proxy-test"  # Default model name, can be changed based on your setup

def send_message(message, model=MODEL_NAME, temperature=0.7, max_tokens=1000):
    """
    Send a message to local LLM server API
    
    Args:
        message (str): The message to send
        model (str): Model name (depends on your LLM server setup)
        temperature (float): Controls randomness (0.0 to 1.0)
        max_tokens (int): Maximum tokens in response
    
    Returns:
        str: The AI response or error message
    """
    
    headers = {
        "Content-Type": "application/json",
        "Authorization": "Bearer test-inf"
    }
    
    data = {
        "model": model,
        "messages": [
            {
                "role": "user",
                "content": message
            }
        ],
        "temperature": temperature,
        "max_tokens": max_tokens,
        "stream": False
    }
    
    response = requests.post(LLM_SERVER_URL, headers=headers, json=data, timeout=60)
    response.raise_for_status()
    
    result = response.json()
    return result["choices"][0]["message"]["content"]

def main():
    """Run in interactive mode for continuous conversation"""
    print("Local LLM Chat Client")
    print("-" * 40)
    
    while True:
        try:
            user_input = input("\nYou: ").strip()
                
            if not user_input:
                continue
                
            print("AI: ", end="", flush=True)
            response = send_message(user_input)
            print(response)
            
        except KeyboardInterrupt:
            print("\nGoodbye!")
            break
        except EOFError:
            print("\nGoodbye!")
            break

if __name__ == "__main__":
    main()