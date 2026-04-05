FROM python:3.9.6
WORKDIR /app
COPY requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt
COPY auth-service/ ./auth-service/
COPY shared/ ./shared/
ENV PYTHONPATH=/app
CMD ["python", "auth-service/main.py"]