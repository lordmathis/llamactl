"""
MkDocs hook to sync content from README.md to docs/index.md
"""
import re
import os


def on_page_markdown(markdown, page, config, **kwargs):
    """Process markdown content before rendering"""
    # Only process the index.md file
    if page.file.src_path != 'index.md':
        return markdown

    # Get the path to README.md (relative to mkdocs.yml)
    readme_path = os.path.join(os.path.dirname(config['config_file_path']), 'README.md')

    if not os.path.exists(readme_path):
        print(f"Warning: README.md not found at {readme_path}")
        return markdown

    try:
        with open(readme_path, 'r', encoding='utf-8') as f:
            readme_content = f.read()
    except Exception as e:
        print(f"Error reading README.md: {e}")
        return markdown

    # Extract headline (the text in bold after the title)
    headline_match = re.search(r'\*\*(.*?)\*\*', readme_content)
    headline = headline_match.group(1) if headline_match else 'Management server for llama.cpp and MLX instances'

    # Extract features section - everything between ## Features and the next ## heading
    features_match = re.search(r'## Features\n(.*?)(?=\n## |\Z)', readme_content, re.DOTALL)
    if features_match:
        features_content = features_match.group(1).strip()
        # Just add line breaks at the end of each line for proper MkDocs rendering
        features_with_breaks = add_line_breaks(features_content)
    else:
        features_with_breaks = "Features content not found in README.md"

    # Replace placeholders in the markdown
    markdown = markdown.replace('{{HEADLINE}}', headline)
    markdown = markdown.replace('{{FEATURES}}', features_with_breaks)

    return markdown


def add_line_breaks(content):
    """Add two spaces at the end of each line for proper MkDocs line breaks"""
    lines = content.split('\n')
    processed_lines = []

    for line in lines:
        if line.strip():  # Only add spaces to non-empty lines
            processed_lines.append(line.rstrip() + '  ')
        else:
            processed_lines.append(line)

    return '\n'.join(processed_lines)