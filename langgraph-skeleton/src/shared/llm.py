"""LLM singleton — Gemini 3 Flash Preview via langchain-google-genai."""

from langchain_google_genai import ChatGoogleGenerativeAI
from dotenv import load_dotenv

load_dotenv()


def get_llm(temperature: float = 0.7) -> ChatGoogleGenerativeAI:
    """Returns a configured Gemini 3 Flash Preview instance."""
    return ChatGoogleGenerativeAI(
        model="gemini-3-flash-preview",
        temperature=temperature,
    )
