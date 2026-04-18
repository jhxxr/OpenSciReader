from PIL import Image

def manual_center_crop(file_path, output_path, crop_ratio=0.65):
    im = Image.open(file_path).convert('RGBA')
    w, h = im.size
    
    new_size = int(min(w, h) * crop_ratio)
    left = (w - new_size) // 2
    top = (h - new_size) // 2
    right = left + new_size
    bottom = top + new_size
    
    im_cropped = im.crop((left, top, right, bottom))
    im_cropped.save(output_path, 'PNG')
    print(f'Done manual center crop to {output_path} with ratio {crop_ratio}')

manual_center_crop(r'C:\Users\24717\.gemini\antigravity\brain\4a902188-8d59-4e57-9b23-f70d6f1ece92\openscireader_app_icon_1776507217891.png', r'e:\0JHX\Project\OpenSciReader\assets\logo.png')
