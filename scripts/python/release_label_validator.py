import requests
import os
import yaml
import sys

with open('.github/release.yml', 'r') as file:
    try:
        release_yaml = yaml.safe_load(file)
        label_pool = []
        for category in release_yaml['changelog']['categories']:
            label_pool.extend(category['labels'])
    except yaml.YAMLError as exc:
        print(exc)

print(f"One of these labels is required on PR: {label_pool}")
token = os.getenv('GITHUB_TOKEN')
repo = os.getenv('REPOSITORY')

response = requests.get(f'https://api.github.com/repos/{repo}/releases/latest', headers={'Authorization': f'token {token}'})
response.raise_for_status()
latest_release = response.json()
latest_release_date = latest_release['created_at']

response = requests.get(f'https://api.github.com/repos/{repo}/pulls?state=closed&sort=updated&direction=desc', headers={'Authorization': f'token {token}'})
response.raise_for_status()
all_closed_prs = response.json()

prs_since_last_release = [
    pr for pr in all_closed_prs
    if pr['merged_at'] is not None and pr['merged_at'] > latest_release_date
]

valid_prs = []
invalid_prs = []
feature_prs = []
for pr in prs_since_last_release:
    labels = [label['name'] for label in pr['labels']]
    common_labels = set(labels).intersection(label_pool)
    if 'kind/feature' in common_labels:
        feature_prs.append(pr['html_url'])
    if len(common_labels) != 1:
        invalid_prs.append(pr['html_url'])
    else:
        valid_prs.append(pr['html_url'])

print("\nThese PRs have exactly one required label:")
print('\n'.join([f"PR: {pr}" for pr in valid_prs]))


if invalid_prs:
    print("\nThese PRs don't have exactly one required label:")
    print('\n'.join([f"PR: {pr}" for pr in invalid_prs]))
    sys.exit(1)

print("\nAll PRs have exactly one required label")

if feature_prs:
    latest_release_name = latest_release['name'].split(".")
    new_release_name = os.getenv('NAME').split(".")
    if latest_release_name[0] == new_release_name[0] and latest_release_name[1] == new_release_name[1]:
        print("\nThese PRs have kind/feature label, but only the patch version number was bumped:\n" + '\n'.join([f"PR: {pr}" for pr in feature_prs]))
        sys.exit(1)

print("\nVersion name is correct")
