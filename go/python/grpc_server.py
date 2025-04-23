from concurrent import futures

import clear_and_smooth_pb2
import clear_and_smooth_pb2_grpc
import grpc
from clear_and_smooth import clear_and_smooth


class ClearAndSmoothService(clear_and_smooth_pb2_grpc.ClearAndSmoothServiceServicer):
    def __init__(self):
        self.request_count = 0

    def ClearAndSmooth(self, request, context):
        try:
            self.request_count += 1
            print(self.request_count)
            
            smoothed_data = {}
            for key, double_list in request.data.items():
                data = list(double_list.values)
                smoothed_values = clear_and_smooth(data)
                smoothed_data[key] = clear_and_smooth_pb2.DoubleList(values=smoothed_values)
            return clear_and_smooth_pb2.ClearAndSmoothResponse(smoothed_data=smoothed_data)
        except Exception as e:
            print(f"Error in ClearAndSmooth: {e}")
            context.set_details(str(e))
            context.set_code(grpc.StatusCode.INTERNAL)
            return clear_and_smooth_pb2.ClearAndSmoothResponse()
def serve():
    server = grpc.server(futures.ThreadPoolExecutor(max_workers=100))
    clear_and_smooth_pb2_grpc.add_ClearAndSmoothServiceServicer_to_server(ClearAndSmoothService(), server)
    server.add_insecure_port('[::]:50051')
    server.start()
    print("gRPC server is running on port 50051...")
    server.wait_for_termination()

if __name__ == "__main__":
    serve()