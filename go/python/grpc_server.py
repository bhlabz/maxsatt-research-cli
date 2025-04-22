from concurrent import futures

import clear_and_smooth_pb2
import clear_and_smooth_pb2_grpc
import grpc
from clear_and_smooth import clear_and_smooth


class ClearAndSmoothService(clear_and_smooth_pb2_grpc.ClearAndSmoothServiceServicer):
    def ClearAndSmooth(self, request, context):
        data = list(request.data)
        smoothed_data = clear_and_smooth(data)  # Ensure this is a callable function
        return clear_and_smooth_pb2.ClearAndSmoothResponse(smoothed_data=smoothed_data)
    
def serve():
    server = grpc.server(futures.ThreadPoolExecutor(max_workers=10))
    clear_and_smooth_pb2_grpc.add_ClearAndSmoothServiceServicer_to_server(ClearAndSmoothService(), server)
    server.add_insecure_port('[::]:50051')
    server.start()
    print("gRPC server is running on port 50051...")
    server.wait_for_termination()

if __name__ == "__main__":
    serve()