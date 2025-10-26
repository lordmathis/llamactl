"""
MkDocs hook to fix line endings for proper rendering.
Automatically adds two spaces at the end of lines that need line breaks.
"""
import re


def on_page_markdown(markdown, page, config, **kwargs):
    """
    Fix line endings in markdown content for proper MkDocs rendering.
    Adds two spaces at the end of lines that need line breaks.
    """
    lines = markdown.split('\n')
    processed_lines = []
    in_code_block = False
    
    for i, line in enumerate(lines):
        stripped = line.strip()
        
        # Track code blocks
        if stripped.startswith('```'):
            in_code_block = not in_code_block
            processed_lines.append(line)
            continue
            
        # Skip processing inside code blocks
        if in_code_block:
            processed_lines.append(line)
            continue
            
        # Skip empty lines
        if not stripped:
            processed_lines.append(line)
            continue
            
        # Skip lines that shouldn't have line breaks:
        # - Headers (# ## ###)
        # - Blockquotes (>)
        # - Table rows (|)
        # - Lines already ending with two spaces
        # - YAML front matter and HTML tags
        # - Standalone punctuation lines
        if (stripped.startswith('#') or 
            stripped.startswith('>') or
            '|' in stripped or
            line.endswith('  ') or
            stripped.startswith('---') or
            stripped.startswith('<') or
            stripped.endswith('>') or
            stripped in ('.', '!', '?', ':', ';', '```', '---', ',')):
            processed_lines.append(line)
            continue
            
        # Add two spaces to lines that end with regular text or most punctuation
        if stripped and not in_code_block:
            processed_lines.append(line.rstrip() + '  ')
        else:
            processed_lines.append(line)
            
    return '\n'.join(processed_lines)