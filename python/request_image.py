from oauthlib.oauth2 import BackendApplicationClient
from requests_oauthlib import OAuth2Session
from credentials import client_id, client_secret
import time
from requests.exceptions import RequestException

def request_image(start_date:str, end_date:str, geometry, width_pixels, height_pixels):
  evalscript = """
    //VERSION=3
    function setup() {
      return {
        input: ["B05", "B08", "B11", "B02", "B04", "B06", "CLD", "SCL"],
        output: {
          id: "default",
          bands: 8,
          sampleType: SampleType.FLOAT32,
        },
      }
    }

    function evaluatePixel(sample) {
      return [sample.B05, sample.B08, sample.B11, sample.B02, sample.B04, sample.B06, sample.CLD, sample.SCL];
    }
  """

  request = {
    "input": {
      "bounds": {
        "geometry": geometry,
      },
      "data": [
        {
          "dataFilter": {
            "timeRange": {
              "from": start_date,
              "to": end_date
            }
          },
          "type": "sentinel-2-l2a"
        }
      ]
    },
    "output": {
      "width": height_pixels,
      "height": width_pixels,
      "responses": [
        {
          "identifier": "default",
          "format": {
            "type": "image/tiff"
          }
        },
      ]
    },
    "evalscript": evalscript,
    "mosaicking":"mostRecent"
  }


  # Create a session
  client = BackendApplicationClient(client_id=client_id)
  oauth = OAuth2Session(client=client)

  # Get token for the session
  oauth.fetch_token(token_url='https://identity.dataspace.copernicus.eu/auth/realms/CDSE/protocol/openid-connect/token',
                            client_secret=client_secret, include_client_id=True)

  url = "https://sh.dataspace.copernicus.eu/api/v1/process"

  retries = 3
  for attempt in range(retries):
    try:
      response = oauth.post(url, json=request)
      if response.status_code == 200:
        break
      else:
        print(f"Attempt {attempt + 1} failed: {response.text}")
    except RequestException as e:
      print(f"Attempt {attempt + 1} failed: {e}")
    time.sleep(2)  # wait for 2 seconds before retrying
  else:
    raise Exception("Failed to request image after 3 attempts")

  # if response.headers.get("content-length"):
  #     raise Exception("Image not found")

  return response.content

