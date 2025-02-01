import json
from bs4 import BeautifulSoup
import re

def clean_html(html_content):
    """
    The most aggressive HTML cleaner this side of the Mississippi! 
    Nukes everything except meaningful text content.
    """
    soup = BeautifulSoup(html_content, 'html.parser')
    for tag in soup.find_all(['script', 'style', 'nav', 'header', 'footer', 
                            'iframe', 'noscript', 'img', 'svg', 'button']):
        tag.decompose()

    for element in soup.find_all(style=True):
        if 'display:none' in element.get('style', '') or 'visibility:hidden' in element.get('style', ''):
            element.decompose()

    suspicious_patterns = re.compile(r'(menu|nav|footer|header|sidebar|banner|ad|cookie|popup|modal)', re.I)
    for element in soup.find_all(class_=suspicious_patterns):
        element.decompose()
    for element in soup.find_all(id=suspicious_patterns):
        element.decompose()

    text_blocks = []
    for elem in soup.find_all(['p', 'div', 'section', 'article', 'li', 'td']):
        # Only keep blocks with substantial text (> 20 chars)
        text = ' '.join(elem.stripped_strings)
        if len(text) > 20:  # If it's shorter than a tweet, it's probably noise
            text_blocks.append(text)

    clean_text = '\n\n'.join(text_blocks)
    clean_text = re.sub(r'\s+', ' ', clean_text)
    clean_text = re.sub(r'\n\s*\n', '\n\n', clean_text)
    return clean_text.strip()



def clean_llm_response(text):
    """
    Cleans up the LLM response while preserving valid JSON structure.
    Returns the cleaned JSON string or raises ValueError if invalid.
    """
    # First, let's strip those pesky markdown code blocks
    if text.startswith('```'):
        # Find the first and last ``` and extract content between them
        start = text.find('\n', text.find('```')) + 1
        end = text.rfind('```')
        if end == -1:  # No closing backticks? No problem!
            text = text[start:]
        else:
            text = text[start:end]
    
    # Remove any leading/trailing whitespace
    text = text.strip()
    
    try:
        # Validate it's actually JSON by parsing it
        parsed = json.loads(text)
        # And get a clean, consistently formatted version
        return json.dumps(parsed, indent=2)
    except json.JSONDecodeError as e:
        # If it's not valid JSON, let's try some cleanup strategies
        
        # Strategy 1: Remove any markdown artifacts that might be breaking our JSON
        text = re.sub(r'(?m)^#.*$', '', text)  # Remove markdown headers
        text = re.sub(r'(?m)^\s*[-*+]\s.*$', '', text)  # Remove list items
        text = re.sub(r'[*_]{1,2}([^*_]+)[*_]{1,2}', r'\1', text)  # Remove bold/italic
        
        # Strategy 2: Fix common JSON syntax issues
        text = re.sub(r',(\s*[}\]])', r'\1', text)  # Remove trailing commas
        text = re.sub(r'"\s*:\s*"([^"]*)"(\s*[,}])', r'": "\1"\2', text)  # Fix quote issues
        
        try:
            # Try parsing again after cleanup
            parsed = json.loads(text)
            return json.dumps(parsed, indent=2)
        except json.JSONDecodeError:
            # If we still can't parse it, return the cleaned text but warn about invalid JSON
            raise ValueError(f"Could not parse response as valid JSON: {str(e)}\nCleaned text: {text}")

