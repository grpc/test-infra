FROM envoyproxy/envoy-dev:239e202a954fa3078a73b23cbc8e7f256fc991cd

COPY envoy-dynamic-ads-full.yaml /etc/envoy/envoy.yaml
RUN chmod go+r /etc/envoy/envoy.yaml
COPY graceful_exit.sh .
RUN chmod go+r graceful_exit.sh

CMD ["/bin/sh","graceful_exit.sh"]
