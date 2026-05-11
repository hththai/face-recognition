import cv2
import os
import glob
from insightface.app import FaceAnalysis
from qdrant_client import QdrantClient
from qdrant_client.models import (
    Distance,
    VectorParams,
    PointStruct
)

# -----------------------------
# 1. Initialize InsightFace
# -----------------------------
app = FaceAnalysis(
    name="buffalo_l",
    providers=['CUDAExecutionProvider', 'CPUExecutionProvider']
)

app.prepare(ctx_id=0, det_size=(640, 640))

# -----------------------------
# 2. Connect to Qdrant
# -----------------------------
client = QdrantClient(
    host="localhost",
    port=6333
)

collection_name = "face_data"

# Create collection if not exists
collections = client.get_collections().collections
existing = [c.name for c in collections]

if collection_name not in existing:
    client.create_collection(
        collection_name=collection_name,
        vectors_config=VectorParams(
            size=512,
            distance=Distance.COSINE
        )
    )

# -----------------------------
# 3. Process Images
# -----------------------------
def process_and_store(folder_path):

    valid_extensions = ('.jpg', '.jpeg', '.png', '.bmp')

    image_paths = [
        f for f in glob.glob(os.path.join(folder_path, '*'))
        if f.lower().endswith(valid_extensions)
    ]

    print(f"Found {len(image_paths)} images")

    point_id = 0

    for img_path in image_paths:

        img = cv2.imread(img_path)

        if img is None:
            continue

        faces = app.get(img)

        for face in faces:

            embedding = face.embedding.tolist()

            # -----------------------------
            # Search Similar Faces
            # -----------------------------
            results = client.query_points(
		collection_name=collection_name,
		query=embedding,
		limit=1
		).points

            # -----------------------------
            # Match Logic
            # -----------------------------
            if results and results[0].score > 0.7:

                print(
                    f"Match found for "
                    f"{os.path.basename(img_path)} "
                    f"(Score: {results[0].score:.4f})"
                )

            else:

                client.upsert(
                    collection_name=collection_name,
                    points=[
                        PointStruct(
                            id=point_id,
                            vector=embedding,
                            payload={
                                "file_path": img_path
                            }
                        )
                    ]
                )

                print(
                    f"New face registered from "
                    f"{os.path.basename(img_path)}"
                )

                point_id += 1


# -----------------------------
# 4. Input Folder
# -----------------------------
#input_folder = r"~/images"
input_folder = os.path.expanduser("~/images")

process_and_store(input_folder)
