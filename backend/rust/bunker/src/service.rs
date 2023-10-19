//! Contains the service trait for the bunker service.

use crate::pb::Result as PbResult;
use crate::pb::{
    EmbeddingResult, GenerateImageRequest, HealthMessage, ModelOptions, PredictOptions, Reply,
    StatusResponse, TokenizationResponse, TranscriptRequest, TranscriptResult, TtsRequest,
};
use async_trait::async_trait;
use tokio_stream::wrappers::ReceiverStream;
use tonic::{Request, Response, Status};

#[async_trait]
pub trait BackendService<T = ReceiverStream<Result<Reply, Status>>> {
    async fn health(&self, request: Request<HealthMessage>) -> Result<Response<Reply>, Status>;
    async fn predict(&self, request: Request<PredictOptions>) -> Result<Response<Reply>, Status>;
    async fn load_model(
        &self,
        request: Request<ModelOptions>,
    ) -> Result<Response<PbResult>, Status>;
    async fn predict_stream(&self, request: Request<PredictOptions>)
        -> Result<Response<T>, Status>; // https://github.com/rust-lang/rust/issues/29661
    async fn embedding(
        &self,
        request: Request<PredictOptions>,
    ) -> Result<Response<EmbeddingResult>, Status>;
    async fn generate_image(
        &self,
        request: Request<GenerateImageRequest>,
    ) -> Result<Response<PbResult>, Status>;
    async fn audio_transcription(
        &self,
        request: Request<TranscriptRequest>,
    ) -> Result<Response<TranscriptResult>, Status>;
    async fn tts(&self, request: Request<TtsRequest>) -> Result<Response<PbResult>, Status>;
    async fn tokenize_string(
        &self,
        request: Request<PredictOptions>,
    ) -> Result<Response<TokenizationResponse>, Status>;
    async fn status(
        &self,
        request: Request<HealthMessage>,
    ) -> Result<Response<StatusResponse>, Status>;
}
