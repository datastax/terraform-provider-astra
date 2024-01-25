# topic ID must be in the form <cluster>:<persistent|non-persistent>://<tenant>/<namespace>/<topic-name><-partition?>

# For example a non-persistent, non-partitioned topic:
terraform import astra_streaming_topic.example pulsar-gcp-uscentral1:non-persistent://mytenant/mynamespace/my-topic

# And a persistent, partitioned topic:
terraform import astra_streaming_topic.example pulsar-gcp-uscentral1:persistent://mytenant/mynamespace/my-topic2-partition

# In addition, a specific topic partition can be imported, but this is usually not the desired use case.
terraform import astra_streaming_topic.example pulsar-gcp-uscentral1:persistent://mytenant/mynamespace/my-topic2-partition-1
