<p align="center">
  <img src="https://capsule-render.vercel.app/api?type=waving&height=350&color=228B22&text=Maxsatt%20Research&reversal=false&textBg=false&fontSize=45&animation=fadeIn&fontColor=FFFFFF&desc=Forest%20Guardian%20Satellite%20Monitoring%20CLI&fontAlign=50&fontAlignY=42&descAlignY=53" style="width: 100%;">
  <h1 align="center">Maxsatt Research CLI</h1>
</p>

<p align="center">
  <a href="#-product">Product</a>&nbsp;&nbsp;&nbsp;|&nbsp;&nbsp;&nbsp;
  <a href="#-stack">Stack</a>&nbsp;&nbsp;&nbsp;|&nbsp;&nbsp;&nbsp;
  <a href="#-objective">Objective</a>&nbsp;&nbsp;&nbsp;|&nbsp;&nbsp;&nbsp;
  <a href="#-structure">Structure</a>&nbsp;&nbsp;&nbsp;|&nbsp;&nbsp;&nbsp;
  <a href="#-prerequisites">Prerequisites</a>&nbsp;&nbsp;&nbsp;|&nbsp;&nbsp;&nbsp;
  <a href="#-execution">Execution</a>&nbsp;&nbsp;&nbsp;|&nbsp;&nbsp;&nbsp;
  <a href="#-commands">Commands</a>&nbsp;&nbsp;&nbsp;|&nbsp;&nbsp;&nbsp;
  <a href="#-data-setup">Data Setup</a>
</p>

<p align="center">
<a href="https://github.com/forest-guardian/maxsatt-research-cli"><img alt="GitHub" src="https://img.shields.io/badge/GitHub-181717?style=for-the-badge&logo=github&logoColor=white"></a>
<a href="https://go.dev"><img alt="Go" src="https://img.shields.io/badge/Go%201.24-00ADD8?style=for-the-badge&logo=go&logoColor=white"></a>
<a href="https://python.org"><img alt="Python" src="https://img.shields.io/badge/Python-3776AB?style=for-the-badge&logo=python&logoColor=white"></a>
<a href="https://grpc.io"><img alt="gRPC" src="https://img.shields.io/badge/gRPC-4285F4?style=for-the-badge&logo=grpc&logoColor=white"></a>
<a href="https://scikit-learn.org"><img alt="Scikit Learn" src="https://img.shields.io/badge/Scikit_Learn-F7931E?style=for-the-badge&logo=scikit-learn&logoColor=white"></a>
<a href="https://pandas.pydata.org"><img alt="Pandas" src="https://img.shields.io/badge/Pandas-150458?style=for-the-badge&logo=pandas&logoColor=white"></a>
<a href="https://numpy.org"><img alt="NumPy" src="https://img.shields.io/badge/NumPy-013243?style=for-the-badge&logo=numpy&logoColor=white"></a>
</p>

<p align = "center">
<b> 🌍 Forest Guardian Satellite Monitoring System | 🛰️ Sentinel Hub Integration | 🤖 ML-Powered Pest Detection </b>
</p>

## 💻 Product

<p></p> 
<p>An advanced forest monitoring and pest detection system that leverages satellite imagery analysis and machine learning to predict and track forest health conditions. The system processes Sentinel satellite data to analyze vegetation indices, detect pest infestations, and monitor deforestation patterns across forest plots over time.</p>

<p>Key capabilities include real-time forest health assessment, pest spread visualization, historical trend analysis, and ML model training for predictive analytics. The CLI provides an intuitive interface for researchers and forest managers to analyze specific plots or entire forests using satellite-derived metrics.</p>

## ⚙ Stack

This project was developed using the following technologies:

### Backend Services
|                                   |                 Technologies                  |                                     |
| :-------------------------------: | :-------------------------------------------: | :---------------------------------: |
|    [Go 1.24](https://go.dev/)     | [gRPC](https://grpc.io/) | [GODAL](https://github.com/airbusgeo/godal) |
| [OAuth2](https://golang.org/x/oauth2) |  [ORB GIS](https://github.com/paulmach/orb)   |     [ProgressBar](https://github.com/schollz/progressbar)      |

### Machine Learning Services
|                                   |                 Technologies                  |                                     |
| :-------------------------------: | :-------------------------------------------: | :---------------------------------: |
|    [Python 3.x](https://python.org/)     | [Scikit-Learn](https://scikit-learn.org/) | [NumPy](https://numpy.org/) |
| [Pandas](https://pandas.pydata.org/) |  [gRPC](https://grpc.io/)   |     [Protobuf](https://protobuf.dev/)      |

### External APIs
- **Sentinel Hub API** - Satellite imagery acquisition
- **Weather APIs** - Historical climate data integration
- **GeoJSON Processing** - Spatial data handling

## 🎯 Objective

To provide comprehensive forest monitoring capabilities through satellite imagery analysis, enabling:

- **Pest Infestation Detection** - Early identification of forest health issues
- **Deforestation Tracking** - Monitor forest cover changes over time  
- **Vegetation Index Analysis** - Track NDVI, PSRI, NDRE, and NDMI metrics
- **Predictive Modeling** - ML-powered forecasting of forest conditions
- **Historical Analysis** - Long-term trend identification and reporting

## 🌌 Structure

The application is organized into distinct services and data processing layers:

- ### **maxsatt-research-cli**

  - **_go-service_**
    - **_cmd_** - Application entry point and CLI initialization
    - **_internal_**
      - **_ui_** - Interactive menu system and user interface handlers
      - **_dataset_** - Data processing, cleaning, and preparation utilities
      - **_sentinel_** - Satellite imagery acquisition and processing
      - **_ml_** - Machine learning model integration and gRPC clients
      - **_spread_** - Pest clustering and deforestation spread analysis
      - **_weather_** - Historical weather data integration
      - **_delivery_** - Business logic orchestration
      - **_notification_** - Discord integration for error reporting
      - **_properties_** - Configuration and environment management
    - **_output_** - Visualization and report generation

  - **_python-service_**
    - **_main.py_** - gRPC server initialization and service routing
    - **_run_model.py_** - ML model execution and prediction pipeline
    - **_clear_and_smooth.py_** - Data smoothing and noise reduction
    - **_reflectance_model.py_** - Satellite reflectance analysis
    - **_climate_group_model.py_** - Climate-based clustering
    - **_pest_clustering_server.py_** - Pest spread analysis service
    - **_plot_pixels_server.py_** - Pixel-level visualization service

  - **_data_**
    - **_geojsons_** - Forest boundary and plot definition files
    - **_images_** - Processed satellite imagery cache
    - **_training_input_** - ML training datasets
    - **_model_** - Trained model files and configurations
    - **_reports_** - Generated analysis reports and accuracy metrics

  - **_tests_** - Test data and validation utilities

## 🔧 Prerequisites

### Software Requirements
- **Go 1.24+** - [Download](https://go.dev/doc/install)
- **Python 3.8+** - [Download](https://python.org/downloads/)
- **pip** - Python package manager

### API Access
- **Sentinel Hub Account** - For satellite imagery access
- **Weather API Keys** - For historical climate data

### System Requirements
- Minimum 8GB RAM (16GB recommended for large datasets)
- 50GB+ free disk space for imagery cache
- Internet connection for API access

## ⏩ Execution

### Environment Setup

1. **Clone the repository**
   ```bash
   git clone https://github.com/forest-guardian/maxsatt-research-cli
   cd maxsatt-research-cli
   ```

2. **Install Python dependencies**
   ```bash
   cd python-service
   pip install -r requirements.txt
   cd ..
   ```

3. **Install Go dependencies**
   ```bash
   cd go-service
   go mod tidy
   cd ..
   ```

4. **Configure environment variables**
   Create a `.env` file in the root directory:
   ```env
   ROOT_PATH=/path/to/maxsatt-research-cli
   GRPC_PORT=50051
   SENTINEL_API_KEY=your_sentinel_api_key
   WEATHER_API_KEY=your_weather_api_key
   DISCORD_WEBHOOK_URL=your_discord_webhook_url
   ```

### Running the Application

**Start both services simultaneously:**
```bash
make run
```

**Manual startup (alternative):**
```bash
# Terminal 1 - Start Python gRPC server
python python-service/main.py --port=50051

# Terminal 2 - Start Go CLI application  
cd go-service/cmd && go run main.go --port=50051
```

The application will display an ASCII banner and present the main menu with 9 available commands.

## 📋 Commands

### 1. **Analyze Pest Infestation in Forest Plot**
**Purpose:** Analyze a specific forest plot for pest infestation on a given date
**Inputs Required:**
- Model selection (from `/data/model/` folder)
- Forest name (corresponding GeoJSON file must exist)
- Plot ID (defined in GeoJSON features)
- Analysis date (YYYY-MM-DD format or "today")

**Process:**
- Downloads Sentinel imagery for the specified date
- Calculates vegetation indices (NDVI, PSRI, NDRE, NDMI)
- Applies ML model for pest detection
- Generates probability maps and visualizations

**Outputs:**
- Infestation probability maps (`.png`)
- Detailed analysis report (`.json`)
- Processed satellite imagery (`.tif`)

---

### 2. **Analyze Pest Infestation in Entire Forest**
**Purpose:** Comprehensive forest-wide analysis for a specific date
**Inputs Required:**
- Model selection
- Forest name
- Analysis date
- Optional: specific plots to analyze

**Process:**
- Processes all plots within the forest boundary
- Aggregates results across the entire forest area
- Generates forest-wide statistics and trends

**Outputs:**
- Forest-wide infestation maps
- Aggregated statistics report
- Plot-by-plot breakdown analysis

---

### 3. **Analyze Forest Plot Indices Over Time**
**Purpose:** Track vegetation health trends using satellite-derived indices
**Inputs Required:**
- Forest name
- Plot ID
- End date for analysis
- Optional: start date (defaults to 1 year prior)

**Process:**
- Downloads historical satellite imagery
- Calculates time series of vegetation indices
- Identifies trends and anomalies
- Generates temporal visualizations

**Outputs:**
- Time series plots (`.png`)
- Index trend analysis report
- Historical data CSV files

---

### 4. **Create New Dataset**
**Purpose:** Generate ML training datasets from processed satellite data
**Inputs Required:**
- Input CSV file name (from `/data/training_input/`)
- Dataset configuration parameters

**Requirements:**
- Input CSV must contain labeled training data
- Required columns: forest, plot, latitude, longitude, indices, labels

**Process:**
- Loads and validates input training data
- Processes satellite imagery for training locations
- Generates feature vectors with weather integration
- Creates balanced training/validation splits

**Outputs:**
- Training dataset (`.csv`) in `/data/model/`
- Dataset summary report
- Feature importance analysis

---

### 5. **Test Model Accuracy**
**Purpose:** Evaluate machine learning model performance
**Inputs Required:**
- Source model for testing
- Training ratio (percentage for training vs validation)

**Process:**
- Splits dataset into training and validation sets
- Trains model on training data
- Evaluates performance on validation set
- Generates comprehensive accuracy metrics

**Outputs:**
- Accuracy analysis report (`.md`) in `/data/reports/`
- Confusion matrix visualization
- Performance metrics (precision, recall, F1-score)
- Training/validation statistics

---

### 6. **View Available Forests**
**Purpose:** List all forests with available GeoJSON boundary files
**Inputs Required:** None

**Process:**
- Scans `/data/geojsons/` directory
- Lists all available forest boundary files
- Shows forest metadata and plot counts

**Outputs:**
- Console display of available forests
- Forest summary information

---

### 7. **View Available Forest Plots**
**Purpose:** List all plots within a specific forest or all forests
**Inputs Required:**
- Optional: specific forest name

**Process:**
- Reads GeoJSON files to extract plot information
- Lists plot IDs and metadata
- Shows plot boundaries and characteristics

**Outputs:**
- Console display of available plots
- Plot metadata and coordinates

---

### 8. **Analyze Deforestation Spread Over Time**
**Purpose:** Track deforestation patterns and pest spread dynamics
**Inputs Required:**
- Forest name
- Plot ID
- End date for analysis
- Days to cluster (temporal grouping parameter)

**Process:**
- Downloads time series satellite imagery
- Applies clustering algorithms to identify change patterns
- Tracks spread velocity and direction
- Generates temporal spread visualizations

**Outputs:**
- Spread pattern animations (`.gif`)
- Cluster analysis results
- Spread velocity measurements
- Temporal change detection maps

---

### 9. **Plot Pixel Values Over Time**
**Purpose:** Visualize pixel-level changes in satellite indices
**Inputs Required:**
- Forest name
- Plot ID  
- End date
- Pixel coordinates (X, Y)

**Process:**
- Extracts pixel time series from satellite imagery
- Plots temporal changes in spectral indices
- Identifies change points and anomalies
- Generates interactive visualizations

**Outputs:**
- Pixel time series plots (`.png`)
- Change point analysis
- Spectral signature evolution charts

## 📁 Data Setup

### Required Directory Structure
```
data/
├── geojsons/          # Forest boundary files (*.geojson)
├── images/            # Cached satellite imagery
├── training_input/    # ML training datasets (*.csv)
├── model/            # Trained model files (*.csv)
├── reports/          # Generated analysis reports
├── result/           # Processing outputs
├── final/            # Final processed datasets
├── delta/            # Temporal change data
└── weather/          # Historical weather cache
```

### GeoJSON File Requirements
- **Naming:** `{forest_name}.geojson`
- **Structure:** FeatureCollection with plot polygons
- **Required Properties:**
  - `plot_id` - Unique identifier for each plot
  - Optional: `name`, `area`, `type`

**Example GeoJSON structure:**
```json
{
  "type": "FeatureCollection",
  "features": [
    {
      "type": "Feature",
      "properties": {
        "plot_id": "P001",
        "name": "North Section"
      },
      "geometry": {
        "type": "Polygon",
        "coordinates": [[[lon1, lat1], [lon2, lat2], ...]]
      }
    }
  ]
}
```

### Training Data Format
Training CSV files should include:
- `forest` - Forest identifier
- `plot` - Plot identifier  
- `latitude`, `longitude` - Coordinates
- `ndvi`, `psri`, `ndre`, `ndmi` - Vegetation indices
- `avg_temperature`, `avg_humidity`, `total_precipitation` - Weather data
- `label` - Classification label (e.g., "healthy", "infested")

## 🔧 Environment Variables

Required environment variables in `.env` file:

```env
# Application Configuration
ROOT_PATH=/absolute/path/to/maxsatt-research-cli
GRPC_PORT=50051

# API Keys
SENTINEL_API_KEY=your_sentinel_hub_api_key
WEATHER_API_KEY=your_weather_api_key

# Notification (Optional)
DISCORD_WEBHOOK_URL=your_discord_webhook_url

# Processing Configuration (Optional)
MAX_CLOUD_COVERAGE=10
IMAGE_RESOLUTION=10
CACHE_ENABLED=true
```

## 🐛 Troubleshooting

### Common Issues

**1. gRPC Connection Errors**
- Ensure both Go and Python services are running
- Check port configuration (default: 50051)
- Verify firewall settings

**2. Missing GeoJSON Files**
```bash
Error: forest 'example' not found
```
- Ensure `example.geojson` exists in `/data/geojsons/`
- Verify GeoJSON format and plot_id properties

**3. API Authentication Errors**
- Verify Sentinel Hub API credentials
- Check API quota and usage limits
- Ensure internet connectivity

**4. Python Dependencies**
```bash
pip install grpcio grpcio-tools numpy scikit-learn pandas python-dotenv
```

**5. Go Module Issues**
```bash
cd go-service && go mod tidy && go mod download
```

### Performance Optimization

- **Large Datasets:** Increase system RAM and use SSD storage
- **API Limits:** Implement caching and batch processing
- **Processing Speed:** Adjust image resolution and temporal range

### Logging and Debugging

The application includes comprehensive error logging:
- Console output with color-coded messages
- Discord notifications for critical errors
- Detailed stack traces for debugging

For additional support, check the application logs and verify all prerequisites are properly installed and configured.