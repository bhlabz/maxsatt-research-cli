import logging
import time
from concurrent import futures

import grpc
import numpy as np
import pest_clustering_pb2
import pest_clustering_pb2_grpc
from sklearn.cluster import DBSCAN
from sklearn.preprocessing import StandardScaler

# Configure logging
logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)

class PestClusteringServicer(pest_clustering_pb2_grpc.PestClusteringServiceServicer):
    
    def ClusterizeSpread(self, request, context):
        """
        Receives an array of delta.Data and returns an array of PestSpreadSample with cluster information.
        """
        logger.info(f"Received ClusterizeSpread request with {len(request.delta_data)} delta data samples")
        
        # Log the received data
        for i, delta_data in enumerate(request.delta_data):
            logger.info(f"Sample {i+1}:")
            logger.info(f"  Farm: {delta_data.farm}")
            logger.info(f"  Plot: {delta_data.plot}")
            logger.info(f"  Position: ({delta_data.x}, {delta_data.y})")
            logger.info(f"  Dates: {delta_data.start_date} to {delta_data.end_date}")
            logger.info(f"  NDRE: {delta_data.ndre:.4f}, NDMI: {delta_data.ndmi:.4f}")
            logger.info(f"  PSRI: {delta_data.psri:.4f}, NDVI: {delta_data.ndvi:.4f}")
            logger.info(f"  Derivatives - NDRE: {delta_data.ndre_derivative:.4f}, NDMI: {delta_data.ndmi_derivative:.4f}")
            logger.info(f"  Derivatives - PSRI: {delta_data.psri_derivative:.4f}, NDVI: {delta_data.ndvi_derivative:.4f}")
            logger.info(f"  Coordinates: ({delta_data.latitude:.6f}, {delta_data.longitude:.6f})")
            if delta_data.label:
                logger.info(f"  Label: {delta_data.label}")
            logger.info("  " + "-" * 50)
        
        # Perform clustering on the data
        pest_spread_samples = self._perform_clustering(request.delta_data)
        
        logger.info(f"Clustering completed. Returning {len(pest_spread_samples)} samples with cluster assignments")
        
        return pest_clustering_pb2.ClusterizeSpreadResponse(
            pest_spread_samples=pest_spread_samples
        )
    
    def _perform_clustering(self, delta_data_list):
        """
        Perform clustering on the delta data and return PestSpreadSample objects.
        """
        if not delta_data_list:
            logger.warning("No delta data provided for clustering")
            return []
        
        # Extract features for clustering
        features = []
        for delta_data in delta_data_list:
            # Use spectral indices and their derivatives as features
            feature_vector = [
                delta_data.ndre,
                delta_data.ndmi,
                delta_data.psri,
                delta_data.ndvi,
                delta_data.ndre_derivative,
                delta_data.ndmi_derivative,
                delta_data.psri_derivative,
                delta_data.ndvi_derivative,
                # delta_data.x,
                # delta_data.y,
            ]
            features.append(feature_vector)
        
        features = np.array(features)
        logger.info(f"Extracted features shape: {features.shape}")
        
        # Normalize features to handle different scales
        scaler = StandardScaler()
        features_normalized = scaler.fit_transform(features)
        logger.info(f"Feature ranges after normalization: min={features_normalized.min(axis=0)}, max={features_normalized.max(axis=0)}")
        
        # Perform DBSCAN clustering with more reasonable parameters
        # eps: maximum distance between two samples for one to be considered as in the neighborhood of the other
        # min_samples: minimum number of samples in a neighborhood for a point to be considered as a core point
        # Using eps=2.0 and min_samples=3 for fewer, more meaningful clusters (targeting ~5 clusters max)
        clustering = DBSCAN(eps=0.5, min_samples=5).fit(features_normalized)
        
        cluster_labels = clustering.labels_
        n_clusters = len(set(cluster_labels)) - (1 if -1 in cluster_labels else 0)
        n_noise = list(cluster_labels).count(-1)
        
        logger.info(f"Clustering results: {n_clusters} clusters found, {n_noise} noise points")
        logger.info(f"Cluster labels: {cluster_labels}")
        
        # Create PestSpreadSample objects
        pest_spread_samples = []
        for i, delta_data in enumerate(delta_data_list):
            sample = pest_clustering_pb2.PestSpreadSample(
                data=delta_data,
                cluster=int(cluster_labels[i])
            )
            pest_spread_samples.append(sample)
            
            logger.info(f"Sample {i+1} assigned to cluster {cluster_labels[i]}")
        
        return pest_spread_samples

def serve_pest_clustering(server):
    """Add pest clustering service to the gRPC server."""
    pest_clustering_pb2_grpc.add_PestClusteringServiceServicer_to_server(
        PestClusteringServicer(), server
    )
    logger.info("Pest Clustering service added to gRPC server")

# Legacy function for standalone execution
def serve():
    """Start the gRPC server."""
    server = grpc.server(futures.ThreadPoolExecutor(max_workers=10))
    serve_pest_clustering(server)
    
    # Listen on port 50052
    listen_addr = '[::]:50052'
    server.add_insecure_port(listen_addr)
    
    logger.info(f"Starting Pest Clustering gRPC server on {listen_addr}")
    server.start()
    
    try:
        # Keep the server running
        while True:
            time.sleep(86400)  # Sleep for 24 hours
    except KeyboardInterrupt:
        logger.info("Shutting down server...")
        server.stop(0)

if __name__ == '__main__':
    serve() 