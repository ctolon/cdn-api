#!/bin/sh

echo "Running update_run.sh..."

file_path="/tmp/clusterID/clusterID"

if [ -f "$file_path" ]; then
  echo "Cluster id file exists."
  CLUSTER_ID=$(cat /tmp/clusterID/clusterID)
  echo "Cluster id: $CLUSTER_ID"
  echo "Removing /tmp/clusterID directory..."
  rm -rf /tmp/clusterID
  echo "Directory removed."
fi

echo "Creating /tmp/clusterID directory..."
mkdir -pv /tmp/clusterID
echo "Directory created."

if [ ! -f "$file_path" ]; then
  #/bin/kafka-storage random-uuid > /tmp/clusterID/clusterID
  echo "${CLUSTER_ID}" > /tmp/clusterID/clusterID
  echo "Cluster id: ${CLUSTER_ID} has been added to the file."
fi

# Docker workaround: Remove check for KAFKA_ZOOKEEPER_CONNECT parameter
sed -i '/KAFKA_ZOOKEEPER_CONNECT/d' /etc/confluent/docker/configure

# Docker workaround: Remove check for KAFKA_ADVERTISED_LISTENERS parameter
sed -i '/dub ensure KAFKA_ADVERTISED_LISTENERS/d' /etc/confluent/docker/configure

# Docker workaround: Ignore cub zk-ready
sed -i 's/cub zk-ready/echo ignore zk-ready/' /etc/confluent/docker/ensure

# KRaft required step: Format the storage directory with a new cluster ID
echo "kafka-storage format --ignore-formatted -t $(cat "$file_path") -c /etc/kafka/kafka.properties" >> /etc/confluent/docker/ensure