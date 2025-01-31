from flask import Flask, request, jsonify
from functools import wraps
import os
import random
from dotenv import load_dotenv
import llm
import db_process
import logging

load_dotenv()
app = Flask(__name__)

# Setup logging - because we're professionals who debug! 
logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)

def require_api_key(f):
    @wraps(f)
    def decorated(*args, **kwargs):
        api_key = request.headers.get('X-API-Key')
        if api_key and api_key == os.getenv('API_KEY'):
            return f(*args, **kwargs)
        return jsonify({"error": "Invalid API key"}), 401
    return decorated

@app.route('/api/process-job', methods=['POST'])
@require_api_key
def process_job():
    try:
        data = request.json
        if not data or 'html_content' not in data:
            return jsonify({"error": "Missing html_content"}), 400

        llm_response = llm.get_llm_response(data['html_content'])
        random_number = random.randint(1, 100)
        # NOTE: not adding to db for now.. lets just focus on this layer
        # job_id = db_process.process_job_posting(llm_response, random_number)

        return jsonify({
            "status": "success",
            "job_id": random_number,
            "parsed_data": llm_response
        })

    except Exception as e:
        logger.error(f"Error processing job: {str(e)}")
        return jsonify({"error": str(e)}), 500

if __name__ == '__main__':
    app.run(host='0.0.0.0', port=5000)
