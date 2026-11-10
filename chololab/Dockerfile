FROM python:3.12-alpine
WORKDIR /app
COPY runner.py .
COPY tests ./tests
CMD ["python", "-m", "unittest", "discover", "-s", "tests", "-v"]
