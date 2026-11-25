FROM python:3.12-alpine

WORKDIR /app
COPY requirements.txt ./requirements.txt
RUN pip install --no-cache-dir -r requirements.txt
COPY app.py ./app.py
COPY tests ./tests

ENV HOST=0.0.0.0
ENV PORT=8080
EXPOSE 8080

USER nobody
CMD ["python", "app.py"]
