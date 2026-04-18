from PIL import Image, ImageDraw

def add_rounded_corners(file_path, output_path):
    im = Image.open(file_path).convert('RGBA')
    w, h = im.size
    
    # Calculate radius: Apple style is typically ~22% of the width
    radius = int(min(w, h) * 0.22)
    
    # Create an anti-aliased mask by drawing it at 4x scale and downsizing
    factor = 4
    mask = Image.new('L', (w * factor, h * factor), 0)
    draw = ImageDraw.Draw(mask)
    draw.rounded_rectangle((0, 0, w * factor, h * factor), radius=radius * factor, fill=255)
    mask = mask.resize((w, h), Image.Resampling.LANCZOS)
    
    # Apply alpha mask
    im.putalpha(mask)
    
    im.save(output_path, 'PNG')
    print(f'Applied rounded corners with radius {radius} (anti-aliased) to {output_path}')

add_rounded_corners(r'e:\0JHX\Project\OpenSciReader\assets\logo.png', r'e:\0JHX\Project\OpenSciReader\assets\logo.png')
