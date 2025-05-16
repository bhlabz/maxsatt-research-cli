import os
import re

import cv2
import hdbscan
import matplotlib.pyplot as plt
import numpy as np
from matplotlib.patches import Circle


def extract_date(filename):
    # Extract date assuming pattern: forest_plot_YYYY-MM-DD.(jpeg|png)
    # Adjust regex if your date format differs
    match = re.search(r'_(\d{4}-\d{2}-\d{2})\.(jpeg|png)$', filename)
    if match:
        return match.group(1)
    else:
        return None


def cluster_and_mark(image_path, min_cluster_size=50, save_path=None):
    image = cv2.imread(image_path)
    if image is None:
        print(f"Failed to load {image_path}")
        return None

    image_rgb = cv2.cvtColor(image, cv2.COLOR_BGR2RGB)
    height, width, _ = image_rgb.shape

    # Prepare features: RGB + normalized XY (0-255)
    X_coords, Y_coords = np.meshgrid(np.arange(width), np.arange(height))
    X_norm = (X_coords / width) * 255
    Y_norm = (Y_coords / height) * 255

    pixels = image_rgb.reshape((-1, 3))
    X_flat = X_norm.flatten()
    Y_flat = Y_norm.flatten()
    features = np.column_stack((pixels, X_flat, Y_flat))

    # Run HDBSCAN with spatial constraints
    clusterer = hdbscan.HDBSCAN(
        min_cluster_size=min_cluster_size,
        cluster_selection_method='leaf',
        metric='manhattan'
    )
    labels = clusterer.fit_predict(features)

    # Calculate cluster centroids (excluding noise)
    centroids = []
    for label in np.unique(labels):
        if label == -1:  # Skip noise
            continue
            
        mask = labels == label
        cluster_points = features[mask]
        
        # Get median RGB and mean position (more robust than mean)
        centroid_color = np.median(cluster_points[:, :3], axis=0).astype(int)
        
        # Denormalize positions
        x_pos = (np.mean(cluster_points[:, 3]) / 255) * width
        y_pos = (np.mean(cluster_points[:, 4]) / 255) * height
        
        centroids.append({
            'position': (x_pos, y_pos),
            'color': centroid_color,
            'size': np.sum(mask)  # Number of pixels in cluster
        })

    # Plot results
    fig, ax = plt.subplots(figsize=(10, 10))
    ax.imshow(image_rgb)
    
    for centroid in centroids:
        x, y = centroid['position']
        color = centroid['color']/255
        size = np.log(centroid['size'])  # Scale marker size logarithmically
        
        circ = Circle(
            (x, y), 
            radius=max(2, size), 
            edgecolor='black',
            facecolor=color,
            linewidth=1,
            alpha=0.7
        )
        ax.add_patch(circ)
    
    ax.axis('off')

    if save_path:
        plt.savefig(save_path, bbox_inches='tight', dpi=150)
        plt.close(fig)
    else:
        plt.show()
        
    return save_path



def process_folder_and_create_video(folder_path, output_folder, video_path, k=5, fps=2):
    if not os.path.exists(output_folder):
        os.makedirs(output_folder)

    # List JPEG files and extract dates
    files = [f for f in os.listdir(folder_path) if f.lower().endswith('.jpeg')]
    files_dates = []
    for f in files:
        date_str = extract_date(f)
        if date_str:
            files_dates.append((f, date_str))
        else:
            print(f"Skipping file with no date: {f}")

    # Sort by date
    files_dates.sort(key=lambda x: x[1])

    saved_images = []
    for filename, date_str in files_dates:
        input_path = os.path.join(folder_path, filename)
        output_filename = f"clustered_{filename.replace('.jpeg', '.png')}"
        output_path = os.path.join(output_folder, output_filename)

        print(f"Processing {filename}...")
        cluster_and_mark(input_path, k=k, save_path=output_path)
        saved_images.append(output_path)

    # Create video from saved images
    if not saved_images:
        print("No images processed, skipping video creation.")
        return
    #todo: create vide 

# Usage example:
folder_path = './images'       # Replace with your folder path
output_folder = './centroids'        # Folder to save images with centroids
video_path = 'pest_spread_video.avi'             # Output video file path

process_folder_and_create_video(folder_path, output_folder, video_path, k=5, fps=2)