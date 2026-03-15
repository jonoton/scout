---
layout: default
title: Detection Config
parent: Configuration
nav_order: 4
---

# Detection Configuration

> 💡 **Tip**
> Filenames for detection logic (e.g., `motion.yaml`, `tensor.yaml`) can be customized in the [Monitor Config](MONITOR).

Scout uses a multi-stage detection pipeline: Motion -> Object -> Face.

> 💡 **Note on Requirements**
> While each detection module is optional, once you enable a module in the [Monitor Config](MONITOR), the fields marked as **Yes** in the tables below become required for that module to function correctly.

## Motion Detection (Optional, `motion.yaml`)

| Field | Type | Required | Default | Description |
| :--- | :--- | :--- | :--- | :--- |
| `skip` | bool | No | `false` | Completely disable motion detection. |
| `padding` | int | No | `0` | Add padding (pixels) around detected motion areas. |
| `scaleWidth` | int | No | `320` | Scale frame width for faster processing. |
| `minPercentage` | int | No | `2` | Min area (%) change for motion. |
| `maxPercentage` | int | No | `75` | Max area (%) change for motion. |
| `thresholdPercent` | int | No | `40` | Sensitivity threshold for pixel changes. |
| `noiseReduction` | int | No | `10` | Filter out small pixel fluctuations. |
| `highlightColor` | string | No | `purple` | Color of the bounding box. |
| `highlightThickness` | int | No | `3` | Thickness of the bounding box. |

## Object Detection (Optional, `tensor.yaml`)

Uses TensorFlow/SSD models to identify specific objects like "person" or "car".

| Field | Type | Required | Default | Description |
| :--- | :--- | :--- | :--- | :--- |
| `skip` | bool | No | `false` | Disable object detection. |
| `forceCpu` | bool | No | `false` | Force CPU processing even if GPU is available. |
| `modelFile` | string | **Yes** | `frozen_inference_graph.pb` | Path to the `.pb` model file. |
| `configFile` | string | No | `ssd_mobilenet_v1...` | Path to the `.pbtxt` or similar config file. |
| `descFile` | string | No | `coco.names` | Path to the labels/names file. |
| `minConfidencePercentage` | int | No | `50` | Minimum confidence (1-100) to consider a match. |
| `allowedList` | list | No | - | List of objects to trigger on (e.g., `person`, `car`). |
| `highlightColor` | string | No | `blue` | Color of the bounding box. |
| `highlightThickness` | int | No | `3` | Thickness of the bounding box. |

## Face Detection (Optional, `face.yaml`)

| Field | Type | Required | Default | Description |
| :--- | :--- | :--- | :--- | :--- |
| `skip` | bool | No | `false` | Disable face detection. |
| `modelFile` | string | **Yes** | `res10_300x300...` | Path to the caffe model or weights. |
| `configFile` | string | No | `deploy.prototxt` | Path to the `.prototxt` config file. |
| `minConfidencePercentage` | int | No | `50` | Minimum confidence for face detection. |
| `highlightColor` | string | No | `green` | Color of the bounding box. |
| `highlightThickness` | int | No | `3` | Thickness of the bounding box. |
