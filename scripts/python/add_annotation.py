import yaml
import sys

if len(sys.argv) != 4:
    print("This script reads the Kubernetes resource yaml file and adds (or modifies, if exists) given annotation. "
          "The modified yaml is printed as an output.")
    exit(1)

filename = sys.argv[1]
key = sys.argv[2]
value = sys.argv[3]

with open(filename, 'r') as file:
    document = yaml.safe_load(file)
    if not ("annotations" in document["metadata"]):
        document["metadata"]["annotations"] = {}
    document["metadata"]["annotations"][key] = value

    print(yaml.dump(document))