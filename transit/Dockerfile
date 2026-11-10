FROM python:3.12-alpine AS runtime
WORKDIR /app
COPY app.py ./app.py
ENV PYTHONUNBUFFERED=1 PORT=8082
EXPOSE 8082
USER nobody
CMD ["python", "app.py"]

FROM python:3.12-alpine AS test
WORKDIR /app
COPY app.py ./app.py
COPY tests ./tests
CMD ["python", "-m", "unittest", "discover", "-s", "tests", "-v"]
