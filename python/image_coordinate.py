import rasterio
import matplotlib.pyplot as plt

def latlon_to_xy(tiff_path, lat, lon):
    with rasterio.open(tiff_path) as dataset:
        # Retrieve dataset bounds and transform
        bounds = dataset.bounds
        transform = dataset.transform

        # Check if latitude and longitude are within the bounds of the image
        if not (bounds.left <= lon <= bounds.right and bounds.bottom <= lat <= bounds.top):
            raise ValueError(f"Latitude {lat} and Longitude {lon} are out of bounds for the image.")

        # Convert geographic coordinates (lon, lat) to pixel coordinates
        col, row = ~transform * (lon, lat)  # Invert transform to get pixel position

        # Convert to integers for pixel coordinates
        col, row = int(col), int(row)

        # Validate pixel coordinates within image dimensions
        if (0 <= col < dataset.width) and (0 <= row < dataset.height):
            return col, row
        else:
            raise ValueError(f"Pixel coordinates ({col}, {row}) are out of image bounds.")

def xy_to_latlon(tiff_path, x, y):
    with rasterio.open(tiff_path) as dataset:
        # Retrieve transform
        transform = dataset.transform

        # Convert pixel coordinates (col, row) to geographic coordinates (lon, lat)
        lon, lat = transform * (x, y)
        return lon, lat  # Return in the order (longitude, latitude)

def plot_pixel_on_image(tiff_path, x, y):
    with rasterio.open(tiff_path) as dataset:
        image = dataset.read(1)  # Read the first band

        plt.figure(figsize=(10, 10))
        plt.imshow(image, cmap='gray')
        plt.scatter([x], [y], color='red', s=100)  # Mark the pixel with a red dot
        plt.title(f'Pixel Coordinates: X: {x}, Y: {y}')
        plt.show()

# Example usage
if __name__ == "__main__":
    tiff_path = 'images/Fazenda_Embay_026/Fazenda_Embay_026_2020-01-01.tif'
    longitude, latitude = -52.287341, -19.886962

    try:
        # Convert latitude/longitude to pixel coordinates
        x, y = latlon_to_xy(tiff_path, latitude, longitude)
        print(f"Pixel Coordinates: X: {x}, Y: {y}")

        # Plot the pixel on the image
        # plot_pixel_on_image(tiff_path, x, y)

        # Convert pixel coordinates back to latitude/longitude
        # lat, lon = xy_to_latlon(tiff_path, x, y)
        # print(f"Recovered Coordinates: Latitude: {lat}, Longitude: {lon}")
    except ValueError as e:
        print(e)