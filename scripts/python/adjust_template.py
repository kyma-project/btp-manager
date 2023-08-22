import yaml
import sys
import os

channel_name = os.environ["CHANNEL"]
doc_url_key = "operator.kyma-project.io/doc-url"
doc_url_value = "https://github.com/kyma-project/btp-manager"

filename = sys.argv[1]

with open(filename, 'r') as file:
    document = yaml.safe_load(file)

    # add documentation annotation
    if not ("annotations" in document["metadata"]):
        document["metadata"]["annotations"] = {}
    document["metadata"]["annotations"][doc_url_key] = doc_url_value

    # adjust the name
    document["metadata"]["name"] = "btp-operator-" + channel_name
    # set the channel
    document["spec"]["channel"] = channel_name

    print(yaml.dump(document))