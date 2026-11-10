FROM python:3.12-alpine AS runtime
WORKDIR /app
COPY app.py ./app.py
ENV PYTHONUNBUFFERED=1 PORT=8081
EXPOSE 8081
USER nobody
CMD ["python", "app.py"]

FROM python:3.12-alpine AS test
WORKDIR /app
COPY app.py ./app.py
COPY tests ./tests
CMD ["python", "-m", "unittest", "discover", "-s", "tests", "-v"]
