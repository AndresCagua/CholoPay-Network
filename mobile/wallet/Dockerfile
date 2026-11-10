FROM ghcr.io/cirruslabs/flutter:3.24.5

WORKDIR /app
COPY pubspec.yaml ./
RUN flutter pub get
COPY . .

CMD ["flutter", "test"]
