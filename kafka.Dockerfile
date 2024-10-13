FROM confluentinc/cp-kafka:7.3.8

USER root

RUN mkdir -pv /tmp/kraft-combined-logs
RUN chmod 777 -R /tmp/kraft-combined-logs

# Add Cluster ID
ARG CLUSTER_ID
ENV CLUSTER_ID=${CLUSTER_ID}

# Create Entrypoint for disable zookeeper and add cluster id
RUN mkdir -pv /entrypoint
COPY ./infra/kafka/update_run.sh /entrypoint/update_run.sh
RUN chmod +x /entrypoint/update_run.sh
RUN chown -R appuser:appuser /entrypoint

# Add jaas and admin config for security
COPY ./infra/kafka/jaas.config /etc/kafka/kraft/jaas.config
COPY ./infra/kafka/admin.config /etc/kafka/kraft/admin.config
RUN chown appuser:appuser /etc/kafka/kraft/jaas.config /etc/kafka/kraft/admin.config

# Run entrypoint script for add cluster id
USER appuser
RUN bash -c /entrypoint/update_run.sh