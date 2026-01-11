from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from google.protobuf.internal import containers as _containers

DESCRIPTOR: _descriptor.FileDescriptor

class HealthRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class HealthResponse(_message.Message):
    __slots__ = ("status", "model_loaded", "device", "backend")
    STATUS_FIELD_NUMBER: _ClassVar[int]
    MODEL_LOADED_FIELD_NUMBER: _ClassVar[int]
    DEVICE_FIELD_NUMBER: _ClassVar[int]
    BACKEND_FIELD_NUMBER: _ClassVar[int]
    status: str
    model_loaded: bool
    device: str
    backend: str
    def __init__(self, status: _Optional[str] = ..., model_loaded: bool = ..., device: _Optional[str] = ..., backend: _Optional[str] = ...) -> None: ...

class EmbeddingRequest(_message.Message):
    __slots__ = ("model", "images")
    MODEL_FIELD_NUMBER: _ClassVar[int]
    IMAGES_FIELD_NUMBER: _ClassVar[int]
    model: str
    images: _containers.RepeatedScalarFieldContainer[bytes]
    def __init__(self, model: _Optional[str] = ..., images: _Optional[_Iterable[bytes]] = ...) -> None: ...

class EmbeddingData(_message.Message):
    __slots__ = ("index", "embedding")
    INDEX_FIELD_NUMBER: _ClassVar[int]
    EMBEDDING_FIELD_NUMBER: _ClassVar[int]
    index: int
    embedding: _containers.RepeatedScalarFieldContainer[float]
    def __init__(self, index: _Optional[int] = ..., embedding: _Optional[_Iterable[float]] = ...) -> None: ...

class EmbeddingResponse(_message.Message):
    __slots__ = ("data", "model", "prompt_tokens", "total_tokens")
    DATA_FIELD_NUMBER: _ClassVar[int]
    MODEL_FIELD_NUMBER: _ClassVar[int]
    PROMPT_TOKENS_FIELD_NUMBER: _ClassVar[int]
    TOTAL_TOKENS_FIELD_NUMBER: _ClassVar[int]
    data: _containers.RepeatedCompositeFieldContainer[EmbeddingData]
    model: str
    prompt_tokens: int
    total_tokens: int
    def __init__(self, data: _Optional[_Iterable[_Union[EmbeddingData, _Mapping]]] = ..., model: _Optional[str] = ..., prompt_tokens: _Optional[int] = ..., total_tokens: _Optional[int] = ...) -> None: ...

class AestheticRequest(_message.Message):
    __slots__ = ("images", "return_distribution")
    IMAGES_FIELD_NUMBER: _ClassVar[int]
    RETURN_DISTRIBUTION_FIELD_NUMBER: _ClassVar[int]
    images: _containers.RepeatedScalarFieldContainer[bytes]
    return_distribution: bool
    def __init__(self, images: _Optional[_Iterable[bytes]] = ..., return_distribution: bool = ...) -> None: ...

class AestheticData(_message.Message):
    __slots__ = ("index", "score", "level", "distribution")
    INDEX_FIELD_NUMBER: _ClassVar[int]
    SCORE_FIELD_NUMBER: _ClassVar[int]
    LEVEL_FIELD_NUMBER: _ClassVar[int]
    DISTRIBUTION_FIELD_NUMBER: _ClassVar[int]
    index: int
    score: float
    level: str
    distribution: _containers.RepeatedScalarFieldContainer[float]
    def __init__(self, index: _Optional[int] = ..., score: _Optional[float] = ..., level: _Optional[str] = ..., distribution: _Optional[_Iterable[float]] = ...) -> None: ...

class AestheticResponse(_message.Message):
    __slots__ = ("data", "model", "backend")
    DATA_FIELD_NUMBER: _ClassVar[int]
    MODEL_FIELD_NUMBER: _ClassVar[int]
    BACKEND_FIELD_NUMBER: _ClassVar[int]
    data: _containers.RepeatedCompositeFieldContainer[AestheticData]
    model: str
    backend: str
    def __init__(self, data: _Optional[_Iterable[_Union[AestheticData, _Mapping]]] = ..., model: _Optional[str] = ..., backend: _Optional[str] = ...) -> None: ...

class MultimodalContent(_message.Message):
    __slots__ = ("text", "image")
    TEXT_FIELD_NUMBER: _ClassVar[int]
    IMAGE_FIELD_NUMBER: _ClassVar[int]
    text: str
    image: bytes
    def __init__(self, text: _Optional[str] = ..., image: _Optional[bytes] = ...) -> None: ...

class MultimodalEmbeddingRequest(_message.Message):
    __slots__ = ("model", "contents")
    MODEL_FIELD_NUMBER: _ClassVar[int]
    CONTENTS_FIELD_NUMBER: _ClassVar[int]
    model: str
    contents: _containers.RepeatedCompositeFieldContainer[MultimodalContent]
    def __init__(self, model: _Optional[str] = ..., contents: _Optional[_Iterable[_Union[MultimodalContent, _Mapping]]] = ...) -> None: ...

class MultimodalEmbeddingItem(_message.Message):
    __slots__ = ("index", "embedding", "type")
    INDEX_FIELD_NUMBER: _ClassVar[int]
    EMBEDDING_FIELD_NUMBER: _ClassVar[int]
    TYPE_FIELD_NUMBER: _ClassVar[int]
    index: int
    embedding: _containers.RepeatedScalarFieldContainer[float]
    type: str
    def __init__(self, index: _Optional[int] = ..., embedding: _Optional[_Iterable[float]] = ..., type: _Optional[str] = ...) -> None: ...

class MultimodalEmbeddingResponse(_message.Message):
    __slots__ = ("embeddings", "model", "input_tokens", "image_tokens")
    EMBEDDINGS_FIELD_NUMBER: _ClassVar[int]
    MODEL_FIELD_NUMBER: _ClassVar[int]
    INPUT_TOKENS_FIELD_NUMBER: _ClassVar[int]
    IMAGE_TOKENS_FIELD_NUMBER: _ClassVar[int]
    embeddings: _containers.RepeatedCompositeFieldContainer[MultimodalEmbeddingItem]
    model: str
    input_tokens: int
    image_tokens: int
    def __init__(self, embeddings: _Optional[_Iterable[_Union[MultimodalEmbeddingItem, _Mapping]]] = ..., model: _Optional[str] = ..., input_tokens: _Optional[int] = ..., image_tokens: _Optional[int] = ...) -> None: ...

class HDBSCANParams(_message.Message):
    __slots__ = ("min_cluster_size", "min_samples", "cluster_selection_epsilon", "cluster_selection_method")
    MIN_CLUSTER_SIZE_FIELD_NUMBER: _ClassVar[int]
    MIN_SAMPLES_FIELD_NUMBER: _ClassVar[int]
    CLUSTER_SELECTION_EPSILON_FIELD_NUMBER: _ClassVar[int]
    CLUSTER_SELECTION_METHOD_FIELD_NUMBER: _ClassVar[int]
    min_cluster_size: int
    min_samples: int
    cluster_selection_epsilon: float
    cluster_selection_method: str
    def __init__(self, min_cluster_size: _Optional[int] = ..., min_samples: _Optional[int] = ..., cluster_selection_epsilon: _Optional[float] = ..., cluster_selection_method: _Optional[str] = ...) -> None: ...

class UMAPParams(_message.Message):
    __slots__ = ("enabled", "n_components", "n_neighbors", "min_dist")
    ENABLED_FIELD_NUMBER: _ClassVar[int]
    N_COMPONENTS_FIELD_NUMBER: _ClassVar[int]
    N_NEIGHBORS_FIELD_NUMBER: _ClassVar[int]
    MIN_DIST_FIELD_NUMBER: _ClassVar[int]
    enabled: bool
    n_components: int
    n_neighbors: int
    min_dist: float
    def __init__(self, enabled: bool = ..., n_components: _Optional[int] = ..., n_neighbors: _Optional[int] = ..., min_dist: _Optional[float] = ...) -> None: ...

class Embedding(_message.Message):
    __slots__ = ("values",)
    VALUES_FIELD_NUMBER: _ClassVar[int]
    values: _containers.RepeatedScalarFieldContainer[float]
    def __init__(self, values: _Optional[_Iterable[float]] = ...) -> None: ...

class ClusteringRequest(_message.Message):
    __slots__ = ("embeddings", "image_ids", "hdbscan_params", "umap_params", "task_id")
    EMBEDDINGS_FIELD_NUMBER: _ClassVar[int]
    IMAGE_IDS_FIELD_NUMBER: _ClassVar[int]
    HDBSCAN_PARAMS_FIELD_NUMBER: _ClassVar[int]
    UMAP_PARAMS_FIELD_NUMBER: _ClassVar[int]
    TASK_ID_FIELD_NUMBER: _ClassVar[int]
    embeddings: _containers.RepeatedCompositeFieldContainer[Embedding]
    image_ids: _containers.RepeatedScalarFieldContainer[int]
    hdbscan_params: HDBSCANParams
    umap_params: UMAPParams
    task_id: int
    def __init__(self, embeddings: _Optional[_Iterable[_Union[Embedding, _Mapping]]] = ..., image_ids: _Optional[_Iterable[int]] = ..., hdbscan_params: _Optional[_Union[HDBSCANParams, _Mapping]] = ..., umap_params: _Optional[_Union[UMAPParams, _Mapping]] = ..., task_id: _Optional[int] = ...) -> None: ...

class ClusterResult(_message.Message):
    __slots__ = ("cluster_id", "image_ids", "avg_probability")
    CLUSTER_ID_FIELD_NUMBER: _ClassVar[int]
    IMAGE_IDS_FIELD_NUMBER: _ClassVar[int]
    AVG_PROBABILITY_FIELD_NUMBER: _ClassVar[int]
    cluster_id: int
    image_ids: _containers.RepeatedScalarFieldContainer[int]
    avg_probability: float
    def __init__(self, cluster_id: _Optional[int] = ..., image_ids: _Optional[_Iterable[int]] = ..., avg_probability: _Optional[float] = ...) -> None: ...

class ClusteringResponse(_message.Message):
    __slots__ = ("clusters", "noise_image_ids", "n_clusters", "params_used")
    class ParamsUsedEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: str
        def __init__(self, key: _Optional[str] = ..., value: _Optional[str] = ...) -> None: ...
    CLUSTERS_FIELD_NUMBER: _ClassVar[int]
    NOISE_IMAGE_IDS_FIELD_NUMBER: _ClassVar[int]
    N_CLUSTERS_FIELD_NUMBER: _ClassVar[int]
    PARAMS_USED_FIELD_NUMBER: _ClassVar[int]
    clusters: _containers.RepeatedCompositeFieldContainer[ClusterResult]
    noise_image_ids: _containers.RepeatedScalarFieldContainer[int]
    n_clusters: int
    params_used: _containers.ScalarMap[str, str]
    def __init__(self, clusters: _Optional[_Iterable[_Union[ClusterResult, _Mapping]]] = ..., noise_image_ids: _Optional[_Iterable[int]] = ..., n_clusters: _Optional[int] = ..., params_used: _Optional[_Mapping[str, str]] = ...) -> None: ...

class ProgressUpdate(_message.Message):
    __slots__ = ("task_id", "status", "progress", "message", "result", "error")
    TASK_ID_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    PROGRESS_FIELD_NUMBER: _ClassVar[int]
    MESSAGE_FIELD_NUMBER: _ClassVar[int]
    RESULT_FIELD_NUMBER: _ClassVar[int]
    ERROR_FIELD_NUMBER: _ClassVar[int]
    task_id: int
    status: str
    progress: int
    message: str
    result: ClusteringResponse
    error: str
    def __init__(self, task_id: _Optional[int] = ..., status: _Optional[str] = ..., progress: _Optional[int] = ..., message: _Optional[str] = ..., result: _Optional[_Union[ClusteringResponse, _Mapping]] = ..., error: _Optional[str] = ...) -> None: ...
