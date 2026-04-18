import sys
try:
    from PIL import Image, ImageChops
except ImportError:
    import subprocess
    subprocess.check_call([sys.executable, '-m', 'pip', 'install', 'Pillow'])
    from PIL import Image, ImageChops

def crop_margins(file_path, output_path):
    im = Image.open(file_path).convert('RGB')
    bg = Image.new(im.mode, im.size, im.getpixel((0, 0)))
    diff = ImageChops.difference(im, bg)
    diff = ImageChops.add(diff, diff, 2.0, -100)
    bbox = diff.getbbox()
    
    if bbox:
        print('Bounding box detected:', bbox)
        padding = 10
        left = max(0, bbox[0] - padding)
        top = max(0, bbox[1] - padding)
        right = min(im.size[0], bbox[2] + padding)
        bottom = min(im.size[1], bbox[3] + padding)
        
        # Keep it square
        width = right - left
        height = bottom - top
        size = max(width, height)
        
        center_x = (left + right) // 2
        center_y = (top + bottom) // 2
        
        new_left = max(0, center_x - size // 2)
        new_top = max(0, center_y - size // 2)
        new_right = min(im.size[0], new_left + size)
        new_bottom = min(im.size[1], new_top + size)
        
        im_cropped = im.crop((new_left, new_top, new_right, new_bottom))
        
        # Convert to RGBA for transparent rounded corners
        im_cropped = im_cropped.convert('RGBA')
        
        im_cropped.save(output_path, 'PNG')
        print(f'Cropped and saved to {output_path}')
    else:
        print('Could not auto-crop. Just doing a center 75% crop.')
        w, h = im.size
        size = int(min(w, h) * 0.75)
        new_left = (w - size) // 2
        new_top = (h - size) // 2
        im_cropped = im.crop((new_left, new_top, new_left + size, new_top + size)).convert('RGBA')
        im_cropped.save(output_path, 'PNG')
        print(f'Center cropped and saved to {output_path}')

crop_margins(r'C:\Users\24717\.gemini\antigravity\brain\4a902188-8d59-4e57-9b23-f70d6f1ece92\openscireader_app_icon_1776507217891.png', r'e:\0JHX\Project\OpenSciReader\assets\logo.png')
