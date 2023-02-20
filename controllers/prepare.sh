#!/bin/bash

k3d cluster delete ttt &&
k3d cluster create ttt &&
kubectl config use-context k3d-ttt