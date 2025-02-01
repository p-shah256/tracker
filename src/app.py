# app.py - Because naming things is hard 😅
from fastapi import FastAPI, Security, Depends
from fastapi.security.api_key import APIKeyHeader
from redis import Redis
from rq import Queue

app = FastAPI()
redis_conn = Redis()
q = Queue(connection=redis_conn)

API_KEY_NAME = "X-API-Key"
api_key_header = APIKeyHeader(name=API_KEY_NAME)

@app.post("/job")
async def process_job(html: str, api_key: str = Depends(api_key_header)):
    # Queue the job
    job = q.enqueue('worker.process_job', html)
    return {"job_id": job.id}
