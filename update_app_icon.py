import shutil
from PIL import Image

logo_path = r'e:\0JHX\Project\OpenSciReader\assets\logo.png'
app_icon_path = r'e:\0JHX\Project\OpenSciReader\build\appicon.png'
windows_icon_path = r'e:\0JHX\Project\OpenSciReader\build\windows\icon.ico'

# Copy to appicon.png
shutil.copy(logo_path, app_icon_path)

# Convert and save to icon.ico
img = Image.open(logo_path).convert('RGBA')
# ICO format needs sizes. Standard sizes for Windows
sizes = [(256, 256), (128, 128), (64, 64), (48, 48), (32, 32), (16, 16)]
try:
    img.save(windows_icon_path, format='ICO', sizes=sizes)
    print(f'Done! Successfully updated appicon.png and windows/icon.ico')
except Exception as e:
    print(f'Failed to save ICO: {e}')
