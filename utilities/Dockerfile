FROM python:3.12-alpine

WORKDIR /app
COPY app.py ./app.py
COPY tests ./tests

ENV HOST=0.0.0.0
ENV PORT=8080
EXPOSE 8080

USER nobody
CMD ["python", "app.py"]
