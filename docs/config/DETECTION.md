---
layout: default
title: Detection
parent: Configuration
nav_order: 4
description: "Tune motion, object, and face detection."
---

# Detection
{: .no_toc }

1. TOC
{:toc}

## How detection works

Each frame goes through three stages. Each stage only runs when the one before it found something, which keeps CPU use low:

```text
Frame ─▶ Motion ─(motion found)─▶ Object ─(object found)─▶ Face
```

1. **Motion** compares the frame with recent frames and marks areas that changed.
2. **Object** runs an SSD MobileNet model (COCO labels such as `person`, `car`, `dog`) on frames with motion. It keeps an object only if it overlaps a motion area, which filters out things like parked cars.
3. **Face** runs a face model on frames that contain an object.

**All three stages are on by default** with the settings shown below. The `motion`, `tensor`, and `face` files are optional and only change those settings. To turn a stage off, set `skip: true` in its file. Object detection is what triggers [event recordings and alerts]({{ site.baseurl }}{% link config/RECORDING_ALERTS.md %}), and it only runs on frames with motion, so turning off motion detection also stops event recordings and alerts. Continuous recording is not affected.

Object and face detection use an NVIDIA GPU automatically when Scout is built with CUDA and one is available.

## Motion detection

Set with the `motion` field of a [camera config]({{ site.baseurl }}{% link config/MONITOR.md %}), for example `motion: motion.yaml`.

| Field | Type | Default | Description |
| :--- | :--- | :--- | :--- |
| `skip` | bool | `false` | Turn motion detection off. |
| `scaleWidth` | int | `320` | Width (pixels) frames are scaled to before processing. Smaller is faster. |
| `minPercentage` | int | `2` | Smallest motion area, as a percentage of the frame. |
| `maxPercentage` | int | `75` | Largest motion area, as a percentage of the frame. |
| `thresholdPercent` | int | `40` | How different a pixel must be to count as changed. Higher is less sensitive. |
| `noiseReduction` | int | `11` | Blur size used to ignore small pixel noise. Must be odd; even values are rounded up. |
| `maxMotions` | int | `20` | Most motion areas in one frame. If a frame has more, all of its motion is ignored, as with `overloadPercent`. |
| `overloadPercent` | int | `90` | If a single motion area covers this percentage of the frame or more (for example, when the camera switches to night mode), motion in that frame is ignored. |
| `minMotionFrames` | int | `1` | Frames in a row an area must move before it counts. Raise this to ignore brief flickers. |
| `backgroundHistory` | int | `500` | Frames of history used to model the background. |
| `backgroundThreshold` | int | `16` | How closely a pixel must match the background model. |
| `detectShadows` | bool | `true` | Detect shadows so they aren't treated as motion. |
| `closingSize` | int | `3` | Size (pixels) used to close small gaps inside motion areas. |
| `mergeOverlapPercent` | int | `20` | Merge two motion areas when they overlap by at least this percentage of the smaller one. |
| `mergeDistance` | int | `10` | Merge motion areas that are within this many pixels of each other. |
| `gridSize` | int | `16` | Grid cell size (pixels) used to track motion between frames. |
| `padding` | int | `0` | Pixels added around each motion box. |
| `highlightColor` | string | `purple` | Box color in highlighted images. |
| `highlightThickness` | int | `3` | Box line thickness. |

{: .tip }
Getting alerts for swaying branches, rain, or sunlight? Raise `minMotionFrames` and `minPercentage` first, then `thresholdPercent`.

## Object detection

Set with the `tensor` field of a camera config, for example `tensor: tensor.yaml`.

| Field | Type | Default | Description |
| :--- | :--- | :--- | :--- |
| `skip` | bool | `false` | Turn object detection off. |
| `allowedList` | list | all labels | Labels that count as detections, such as `person` and `car`. Empty means every [COCO label](https://github.com/jonoton/scout/blob/master/data/tensor/coco.names). |
| `minConfidencePercentage` | int | `50` | Lowest model confidence (1 to 100) to accept a detection. |
| `priorityList` | list | `person` at `50` | Which label wins when different objects overlap, with an optional confidence threshold each. See below. |
| `minMotionFrames` | int | `1` | Frames in a row with motion before object detection runs. |
| `minPercentage` | int | `2` | Smallest object, as a percentage of the frame. |
| `maxPercentage` | int | `50` | Largest object, as a percentage of the frame. |
| `minOverlapPercentage` | int | `75` | How much of the object box must overlap a motion area. Objects that aren't moving are ignored. |
| `sameOverlapPercentage` | int | `85` | Overlap at which two boxes are treated as the same object. |
| `scaleWidth` | int | `320` | Width (pixels) frames are scaled to before detection. |
| `forceCpu` | bool | `false` | Use the CPU even when a CUDA GPU is available. |
| `modelFile` | string | `frozen_inference_graph.pb` | Model weights. |
| `configFile` | string | `ssd_mobilenet_v1_coco_2017_11_17.pbtxt` | Model config. |
| `descFile` | string | `coco.names` | Label names. |
| `padding` | int | `0` | Pixels added around each object box. |
| `highlightColor` | string | `blue` | Box color in highlighted images. |
| `highlightThickness` | int | `3` | Box line thickness. |

Model files are looked up in `data/tensor/` and then `.data/tensor/` next to the Scout executable, or can be absolute paths. The Docker image and the repository already include the defaults.

### Priority list

When boxes for different labels overlap (for example a `person` and a `chair`), Scout usually keeps the one with the higher confidence. A label listed earlier in `priorityList` can also replace a later or unlisted one, as long as it meets its own confidence threshold:

```yaml
priorityList:
  - description: person
    minConfidencePercentage: 70
  - description: car
    minConfidencePercentage: 60
```

### Example

```yaml
allowedList:
  - person
  - car
  - truck
minConfidencePercentage: 60
```

## Face detection

Set with the `face` field of a camera config, for example `face: face.yaml`. Face detection runs on frames with at least one detected object, and only faces that overlap an object are kept.

| Field | Type | Default | Description |
| :--- | :--- | :--- | :--- |
| `skip` | bool | `false` | Turn face detection off. |
| `minConfidencePercentage` | int | `50` | Lowest model confidence (1 to 100) to accept a face. |
| `maxPercentage` | int | `50` | Largest face, as a percentage of the frame. |
| `minOverlapPercentage` | int | `75` | How much of the face box must overlap a detected object. |
| `scaleWidth` | int | `320` | Width (pixels) frames are scaled to before detection. |
| `forceCpu` | bool | `false` | Use the CPU even when a CUDA GPU is available. |
| `modelFile` | string | `res10_300x300_ssd_iter_140000.caffemodel` | Model weights. |
| `configFile` | string | `deploy.prototxt` | Model config. |
| `padding` | int | `0` | Pixels added around each face box. |
| `highlightColor` | string | `green` | Box color in highlighted images. |
| `highlightThickness` | int | `3` | Box line thickness. |

Model files are looked up in `data/face/` and then `.data/face/`, the same way as object detection.
