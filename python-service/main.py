import argparse
import os
from concurrent import futures
from datetime import datetime

import clear_and_smooth_pb2
import clear_and_smooth_pb2_grpc
import grpc
import pandas as pd
import run_model_pb2
import run_model_pb2_grpc
from clear_and_smooth import clear_and_smooth
from dotenv import load_dotenv
from run_model import run_model
from pest_clustering_server import serve_pest_clustering


class ClearAndSmoothService(clear_and_smooth_pb2_grpc.ClearAndSmoothServiceServicer):
    def __init__(self):
        pass

    def ClearAndSmooth(self, request, context):
        try:
            smoothed_data = {}
            for key, double_list in request.data.items():
                data = list(double_list.values)
                smoothed_values = clear_and_smooth(data)
                smoothed_data[key] = clear_and_smooth_pb2.DoubleList(values=smoothed_values)
            return clear_and_smooth_pb2.ClearAndSmoothResponse(smoothed_data=smoothed_data)
        except Exception as e:
            # print(f"Error in ClearAndSmooth: {e}")
            context.set_details(str(e))
            context.set_code(grpc.StatusCode.INTERNAL)
            return clear_and_smooth_pb2.ClearAndSmoothResponse()
        
class RunModelServiceServicer(run_model_pb2_grpc.RunModelServiceServicer):
    def RunModel(self, request, context):
        try: 
            rows = []
            for item in request.data:
                weather = item.weather
                delta = item.delta
                row = {
                    "avg_temperature": weather.avg_temperature,
                    "temp_std_dev": weather.temp_std_dev,
                    "avg_humidity": weather.avg_humidity,
                    "humidity_std_dev": weather.humidity_std_dev,
                    "total_precipitation": weather.total_precipitation,
                    "dry_days_consecutive": weather.dry_days_consecutive,
                    "farm": delta.farm,
                    "plot": delta.plot,
                    "delta_min": delta.delta_min,
                    "delta_max": delta.delta_max,
                    "delta": delta.delta,
                    "start_date": delta.start_date,
                    "end_date": delta.end_date,
                    "latitude": delta.latitude,
                    "longitude": delta.longitude,
                    "x": delta.x,
                    "y": delta.y,
                    "ndre": getattr(delta, "ndre", None), 
                    "ndmi": getattr(delta, "ndmi", None),
                    "psri": delta.psri,
                    "ndvi": delta.ndvi,
                    "ndre_derivative": getattr(delta, "ndre_derivative", None),
                    "ndmi_derivative": getattr(delta, "ndmi_derivative", None),
                    "psri_derivative": delta.psri_derivative,
                    "ndvi_derivative": delta.ndvi_derivative,
                    "label": getattr(delta, "label", None),
                    "created_at": datetime.now().isoformat(),
                }
                rows.append(row)

            # Create a DataFrame
            df = pd.DataFrame(rows)
            # print(df)
            print(f"Running model: {request.model}")
            result = run_model(request.model,df)
            response = run_model_pb2.RunModelResponse()
            for item in result:
                pixel_result = run_model_pb2.PixelResult(
                    x=item['x'],
                    y=item['y'],
                    latitude=item['latitude'],
                    longitude=item['longitude'],
                    result=[
                        run_model_pb2.LabelProbability(
                            label=label_prob['label'],
                            probability=label_prob['probability']
                        ) for label_prob in item['result']
                    ]
                )
                response.results.append(pixel_result)
            return response
        except Exception as e:
            print(f"Error in RunModel: {e}")
            context.set_details(str(e))
            context.set_code(grpc.StatusCode.INTERNAL)
            return run_model_pb2.RunModelResponse()

    
def serve(port):
    server = grpc.server(futures.ThreadPoolExecutor(max_workers=200),options=[
        ('grpc.max_send_message_length', 10 * 1024 * 1024),  # 10 MB
        ('grpc.max_receive_message_length', 10 * 1024 * 1024)
    ])
    clear_and_smooth_pb2_grpc.add_ClearAndSmoothServiceServicer_to_server(ClearAndSmoothService(), server)
    run_model_pb2_grpc.add_RunModelServiceServicer_to_server(RunModelServiceServicer(), server)
    serve_pest_clustering(server)  # Add pest clustering service
    server.add_insecure_port(f'[::]:{port}')
    server.start()
    # print("gRPC server is running on port 50051...")
    server.wait_for_termination()

if __name__ == "__main__":
    load_dotenv(os.path.join(os.path.dirname(__file__), "../.env"))
    parser = argparse.ArgumentParser(description="Run the gRPC server.")
    parser.add_argument("--port", type=int, default=50051, help="Port to run the gRPC server on.")
    args = parser.parse_args()
    serve(args.port)