import grpc
import gam_smoothing_pb2
import gam_smoothing_pb2_grpc

def run():
    with grpc.insecure_channel('localhost:50051') as channel:
        stub = gam_smoothing_pb2_grpc.GamSmoothingServiceStub(channel)
        response = stub.Smooth(gam_smoothing_pb2.SmoothRequest(values=[1, 2, 3, 4, 5], lam=0.1))
        print("Smoothed Values:", response.smoothed_values)

if __name__ == '__main__':
    run()