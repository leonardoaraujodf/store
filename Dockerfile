FROM ubuntu:24.04

RUN apt-get update && apt-get install -y \
    build-essential \
    cmake \
    libmysqlcppconn-dev \
    libboost-all-dev \
    libssl-dev \
    wget \
    curl \
    git \
    && apt-get clean

# Install Crow (header-only library)
RUN mkdir -p /usr/local/include/crow
RUN wget -O /usr/local/include/crow/crow_all.h https://github.com/CrowCpp/Crow/releases/download/v0.3/crow_all.h

# Create a directory for the application
WORKDIR /app

# Copy the current directory contents into the container at /app
COPY . /app

# Build the application
RUN mkdir build && cd build && cmake .. && make

# Expose the port the app runs on
EXPOSE 18080

# Run the application
CMD ["./build/OnlineStore"]
