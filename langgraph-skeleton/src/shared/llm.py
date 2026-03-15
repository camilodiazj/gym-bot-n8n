"""LLM singleton — Gemini 2.0 Flash via langchain-google-genai."""

from langchain_google_genai import ChatGoogleGenerativeAI
from dotenv import load_dotenv

load_dotenv()


def get_llm(temperature: float = 0.7) -> ChatGoogleGenerativeAI:
    """Returns a configured Gemini 2.0 Flash instance."""
    return ChatGoogleGenerativeAI(
        model="gemini-2.0-flash",
        temperature=temperature,
    )
