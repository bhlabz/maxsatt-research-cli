import numpy as np
import matplotlib.pyplot as plt
from sklearn.cluster import DBSCAN
from scipy.spatial.distance import cdist

# --- CONFIGURATION PARAMETERS ---
EPSILON = 3       # Maximum distance between pixels for DBSCAN
MIN_SAMPLES = 3   # Minimum infested pixels to form a cluster

def detect_clusters(data):
    """
    Clusters infested pixels.
    If data is a list of dicts, it filters pixels where pest_prob > 50.
    If data is a NumPy array, it is assumed to be an array of coordinates.
    Returns a tuple (labels, pixels).
    """
    if isinstance(data, np.ndarray):
        pixels = data
    else:
        pixels = np.array([[p["x"], p["y"]] for p in data if p.get("pest_prob", 0) > 50])
    if pixels.size == 0:
        return [], pixels
    clustering = DBSCAN(eps=EPSILON, min_samples=MIN_SAMPLES).fit(pixels)
    return clustering.labels_, pixels

def compute_state_metrics(image, img_size):
    """
    Computes summary metrics for an image:
      - Centroid of infested pixels.
      - Density (ratio of infested pixels to image area).
      - Edge ratio (proportion of infested pixels near edges).
      - Total number of infested pixels.
    """
    _, pixels = detect_clusters(image)
    if pixels.size == 0:
        return {"centroid": None, "density": 0, "edge_ratio": 0, "num_pixels": 0}
    
    centroid = np.mean(pixels, axis=0)
    num_pixels = len(pixels)
    density = num_pixels / (img_size[0] * img_size[1])
    
    edge_threshold = 10  # pixels within this distance from the edge are considered "edge" pixels
    edge_pixels = [p for p in pixels if (p[0] < edge_threshold or p[1] < edge_threshold or 
                                           p[0] > img_size[0] - edge_threshold or p[1] > img_size[1] - edge_threshold)]
    edge_ratio = len(edge_pixels) / num_pixels
    
    return {"centroid": centroid, "density": density, "edge_ratio": edge_ratio, "num_pixels": num_pixels}

def analyze_spread(previous_state, current_state, img_size):
    """
    Computes percentage contributions for different spread patterns by comparing
    current state's metrics to previous state's metrics.
    """
    if previous_state["centroid"] is None or current_state["centroid"] is None:
        return {"Edges Inward": 0, "Directional Spread": 0, "Homogeneous Spread": 0, "Focal Expansion": 0}
    
    # Directional Spread: computed as movement of the centroid relative to image size.
    delta = np.linalg.norm(current_state["centroid"] - previous_state["centroid"])
    directional_score = min(delta / max(img_size), 1.0)
    
    # Edges Inward: if the edge ratio increases (i.e., more infestation near edges).
    delta_edge = current_state["edge_ratio"] - previous_state["edge_ratio"]
    edges_inward_score = max(delta_edge, 0)
    
    # Homogeneous Spread: change in density (scaled arbitrarily; tune as needed).
    delta_density = abs(current_state["density"] - previous_state["density"])
    homogeneous_score = min(delta_density * 10, 1.0)
    
    # Focal Expansion: a placeholder score; here we boost the score if the number of infested pixels increases.
    focal_expansion_score = 0.2 if current_state["num_pixels"] > previous_state["num_pixels"] else 0
    
    # Normalize scores so that the sum becomes 100%
    total = directional_score + edges_inward_score + homogeneous_score + focal_expansion_score
    if total == 0:
        return {"Edges Inward": 0, "Directional Spread": 0, "Homogeneous Spread": 0, "Focal Expansion": 0}
    
    return {
        "Edges Inward": round(edges_inward_score * 100 / total, 0),
        "Directional Spread": round(directional_score * 100 / total, 0),
        "Homogeneous Spread": round(homogeneous_score * 100 / total, 0),
        "Focal Expansion": round(focal_expansion_score * 100 / total, 0)
    }

def sort_images_by_date(image_data):
    """Sorts the image dictionary by date keys."""
    return {date: image_data[date] for date in sorted(image_data.keys())}

def analyze_infestation(image_data, img_size):
    """
    Processes images in chronological order.
    Maintains a 'state' summary from previous images to compare with current.
    Returns a dictionary mapping date to spread analysis percentages.
    """
    sorted_images = sort_images_by_date(image_data)
    dates = list(sorted_images.keys())
    spread_results = {}
    
    # Initialize state with no infestation.
    previous_state = {"centroid": None, "density": 0, "edge_ratio": 0, "num_pixels": 0}
    
    for date in dates:
        print(f"Processing date: {date}")
        current_state = compute_state_metrics(sorted_images[date], img_size)
        spread_results[date] = analyze_spread(previous_state, current_state, img_size)
        # Update the state only if current image shows infestation.
        if current_state["num_pixels"] > 0:
            previous_state = current_state

    return spread_results

def plot_evolution(image_data, spread_results, img_size):
    """
    Plots the evolution of infestation:
      - Each subplot corresponds to a date.
      - Infested pixels are shown as red dots.
      - The computed centroid is overlaid as a blue "x".
      - The title shows the date and spread analysis percentages.
    """
    sorted_images = sort_images_by_date(image_data)
    dates = list(sorted_images.keys())
    n_dates = len(dates)
    
    fig, axes = plt.subplots(1, n_dates, figsize=(5 * n_dates, 5))
    if n_dates == 1:
        axes = [axes]
    
    for ax, date in zip(axes, dates):
        _, pixels = detect_clusters(sorted_images[date])
        # Plot infested pixels
        if pixels.size > 0:
            ax.scatter(pixels[:, 0], pixels[:, 1], c='red', label='Infested Pixels')
        else:
            ax.text(0.5, 0.5, "No Infestation", ha='center', va='center', transform=ax.transAxes)
        
        # Plot centroid if available.
        state = compute_state_metrics(sorted_images[date], img_size)
        if state["centroid"] is not None:
            ax.scatter(state["centroid"][0], state["centroid"][1], marker='x', color='blue', s=100, label='Centroid')
        
        # Set plot parameters.
        ax.set_title(f"{date}\n{spread_results.get(date, {})}")
        ax.set_xlim(0, img_size[0])
        ax.set_ylim(0, img_size[1])
        ax.invert_yaxis()  # Adjust coordinate system if needed.
        ax.legend()
    
    plt.tight_layout()
    plt.show()

# --- EXAMPLE USAGE ---
if __name__ == "__main__":
    image_data = {
        "2024-01-01": [
            {"x": 2, "y": 2, "healthy_prob": 90, "pest_prob": 10},
            {"x": 5, "y": 5, "healthy_prob": 40, "pest_prob": 60}
        ],
        "2024-01-10": [
            {"x": 2, "y": 2, "healthy_prob": 80, "pest_prob": 20},
            {"x": 3, "y": 3, "healthy_prob": 40, "pest_prob": 60},
            {"x": 6, "y": 6, "healthy_prob": 20, "pest_prob": 80}
        ],
        "2024-01-20": [
            {"x": 1, "y": 1, "healthy_prob": 10, "pest_prob": 90},
            {"x": 2, "y": 2, "healthy_prob": 20, "pest_prob": 80},
            {"x": 3, "y": 3, "healthy_prob": 10, "pest_prob": 90},
            {"x": 7, "y": 7, "healthy_prob": 10, "pest_prob": 90}
        ]
    }
    
    img_size = (10, 10)  # Image dimensions (width, height)
    
    spread_results = analyze_infestation(image_data, img_size)
    for date, results in spread_results.items():
        print(f"\nDate: {date}")
        print(results)
    
    # Plot evolution to visually verify the spread over time.
    plot_evolution(image_data, spread_results, img_size)
