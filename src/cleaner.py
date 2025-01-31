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
    Takes your LLM's markdown soup and turns it into clean text.
    Because who needs formatting when you've got pure content? 😎
    """

    text = re.sub(r'```[\s\S]*?```', '', text)
    text = re.sub(r'(?m)^    .*$', '', text)
    text = re.sub(r'#{1,6}\s.*$', '', text, flags=re.MULTILINE)
    text = re.sub(r'^\s*[-*+]\s', '', text, flags=re.MULTILINE)
    text = re.sub(r'^\s*\d+\.\s', '', text, flags=re.MULTILINE)
    text = re.sub(r'[*_]{1,2}([^*_]+)[*_]{1,2}', r'\1', text)
    text = re.sub(r'\n{3,}', '\n\n', text)
    text = re.sub(r'[\[\](){}`>#]', '', text)

    return text.strip()
