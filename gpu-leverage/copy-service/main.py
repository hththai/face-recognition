import os
import shutil
import numpy as np
from qdrant_client import QdrantClient
from sklearn.cluster import DBSCAN # Or use cuml if you installed it

client = QdrantClient(host="localhost", port=6333)
collection_name = "face_data"

# CHANGE: Now pointing to a local folder in your current directory
output_base_dir = os.path.join(os.getcwd(), "folder-output")

# 1. Pull data from Qdrant
result = client.scroll(collection_name=collection_name, with_vectors=True, with_payload=True, limit=5000)
points = result[0]

if not points:
    print("No data found in Qdrant!")
    exit()

# 2. Extract Vectors
vectors = np.array([p.vector for p in points], dtype='float32')

print(f"Clustering {len(vectors)} faces on RTX 5080...")

# 3. Clustering
# eps 0.35 is the 'strictness' threshold. Adjust as needed.
model = DBSCAN(eps=0.45, min_samples=3, metric='cosine')
labels = model.fit_predict(vectors)

# 4. Organize into local folders
print(f"Starting organization of {len(labels)} faces...")

# Track counts
counts = {"success": 0, "error": 0}

for i, label in enumerate(labels):
    payload = points[i].payload
    source_path = payload.get("file_path")

    if not source_path:
        continue

    # Create folder name
    folder_name = f"Person_{label}" if label != -1 else "Unknown"
    target_dir = os.path.join(output_base_dir, folder_name)
    os.makedirs(target_dir, exist_ok=True)

    file_name = os.path.basename(source_path)
    dest_path = os.path.join(target_dir, f"face_{i}_{file_name}")

    # --- ADDED PROGRESS LOGGING ---
    print(f"[{i+1}/{len(labels)}] Moving {file_name} to {folder_name}...", end="\r")

    try:
        shutil.copyfile(source_path, dest_path)
        counts["success"] += 1
    except Exception as e:
        print(f"\n[!] Error copying {file_name}: {e}")
        counts["error"] += 1

print(f"\n\nFinished!")
print(f"Total Processed: {len(labels)}")
print(f"Successfully Copied: {counts['success']}")
print(f"Failed: {counts['error']}")
