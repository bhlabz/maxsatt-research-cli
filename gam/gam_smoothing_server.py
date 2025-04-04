import grpc
from concurrent import futures
import numpy as np
from pygam import LinearGAM, s
import gam_smoothing_pb2
import gam_smoothing_pb2_grpc

def gam_smoothing(values, lam):
    x = np.arange(len(values)).reshape(-1, 1)
    y = np.array(values)
    
    if np.all(y == 0) or np.std(y) == 0:
        return y.tolist()

    gam = LinearGAM(s(0, lam=lam)).fit(x, y)
    y_pred = gam.predict(x)
    
    return y_pred.tolist()

class GamSmoothingService(gam_smoothing_pb2_grpc.GamSmoothingServiceServicer):
    def Smooth(self, request, context):
        smoothed_values = gam_smoothing(request.values, request.lam)
        return gam_smoothing_pb2.SmoothResponse(smoothed_values=smoothed_values)

def serve():
    server = grpc.server(futures.ThreadPoolExecutor(max_workers=10))
    gam_smoothing_pb2_grpc.add_GamSmoothingServiceServicer_to_server(GamSmoothingService(), server)
    server.add_insecure_port('[::]:50051')
    server.start()
    print("Server started on port 50051")
    server.wait_for_termination()

if __name__ == '__main__':
    serve()
