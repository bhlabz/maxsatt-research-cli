import os
from collections import Counter
from PIL import Image

def count_pixel_colors(folder_path):
    for filename in os.listdir(folder_path):
        if filename.lower().endswith(".jpeg"):
            image_path = os.path.join(folder_path, filename)
            image = Image.open(image_path).convert("RGB")
            pixels = list(image.getdata())
            color_counts = Counter(pixels)

            print(f"\nImage: {filename}")
            for color, count in color_counts.items():
                print(f"Color {color}: {count} pixels")

            # Specific counts for white and black
            print("White pixels:", color_counts.get((255, 255, 255), 0))
            print("Black pixels:", color_counts.get((0, 0, 0), 0))
            # green, red and blue counts
            print("Green pixels:", color_counts.get((0, 255, 0), 0))
            print("Red pixels:", color_counts.get((255, 0, 0), 0))
            print("Blue pixels:", color_counts.get((0, 0, 255), 0))

# Example usage:
count_pixel_colors("/Users/gabihert/Documents/Projects/forest-guardian/forest-guardian-api-poc/data/result/Fazendas_Manulife_Gema/GMA-025/index/images")