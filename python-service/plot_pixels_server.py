from concurrent import futures

import grpc
import matplotlib
import plot_pixels_pb2
import plot_pixels_pb2_grpc

matplotlib.use('Agg')
import matplotlib.pyplot as plt


class PlotPixelsServicer(plot_pixels_pb2_grpc.PlotPixelsServiceServicer):
    def PlotPixels(self, request, context):
        for pixel in request.pixels:
            ndvi_values = []
            ndre_values = []
            ndmi_values = []
            dates = []

            for date, val in sorted(request.NDVI.items()):
                dates.append(date)
                ndvi_values.append(val)
            for date, val in sorted(request.NDRE.items()):
                ndre_values.append(val)
            for date, val in sorted(request.NDMI.items()):
                ndmi_values.append(val)

            plt.figure()
            plt.plot(dates, ndvi_values, label='NDVI')
            plt.plot(dates, ndre_values, label='NDRE')
            plt.plot(dates, ndmi_values, label='NDMI')
            plt.xlabel('Date')
            plt.ylabel('Value')
            plt.title(f'{request.forest_name} {request.plot_id} Pixel ({pixel.x}, {pixel.y}) Time Series')
            plt.legend()
            plt.xticks(rotation=45, ha='right')
            plt.tight_layout()
            
            # Create a directory for plots if it doesn't exist
            plots_dir = f"../data/result/{request.forest_name}/{request.plot_id}/plots"
            import os
            if not os.path.exists(plots_dir):
                os.makedirs(plots_dir)

            # Generate a unique filename for the plot
            import time
            filename = f"{plots_dir}/pixel_{pixel.x}_{pixel.y}_{int(time.time())}.png"
            plt.savefig(filename)
            plt.close() # Close the plot to free memory

        return plot_pixels_pb2.PlotPixelsResponse(message=f'Plot generated successfully: {filename}')

def serve_plot_pixels(server):
    plot_pixels_pb2_grpc.add_PlotPixelsServiceServicer_to_server(PlotPixelsServicer(), server)