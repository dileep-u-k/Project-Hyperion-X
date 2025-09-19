use actix_web::{get, App, HttpResponse, HttpServer, Responder};
use serde::Serialize;
use std::fs;
use tracing::{info, Level};
use tracing_subscriber::EnvFilter;

#[derive(Serialize, Clone, Default)]
struct Metrics {
    node_name: String,
    cpu_usage_pct: f64,
    mem_usage_pct: f64,
    gpu_count: u32,
}

// Simple, robust metric collection for Phase 1
fn read_cpu_usage_pct() -> f64 {
    let load = fs::read_to_string("/host/proc/loadavg").unwrap_or_default();
    let parts: Vec<&str> = load.split_whitespace().collect();
    let one_min: f64 = parts.get(0).and_then(|s| s.parse().ok()).unwrap_or(0.0);
    let cores: usize = num_cpus::get();
    ((one_min / cores as f64) * 100.0).clamp(0.0, 100.0)
}

fn read_mem_usage_pct() -> f64 {
    let meminfo = fs::read_to_string("/host/proc/meminfo").unwrap_or_default();
    let mut total_kb = 0.0;
    let mut avail_kb = 0.0;
    for line in meminfo.lines() {
        if line.starts_with("MemTotal:") {
            total_kb = line.split_whitespace().nth(1).and_then(|v| v.parse().ok()).unwrap_or(0.0);
        } else if line.starts_with("MemAvailable:") {
            // **THE FIX IS HERE**
            // Corrected the typo from 'avail_.kb' to 'avail_kb'
            avail_kb = line.split_whitespace().nth(1).and_then(|v| v.parse().ok()).unwrap_or(0.0);
        }
    }
    if total_kb > 0.0 { ((total_kb - avail_kb) / total_kb) * 100.0 } else { 0.0 }
}

fn read_gpu_count() -> u32 {
    // Best-effort GPU count for Phase 1
    match fs::read_dir("/host/proc/driver/nvidia/gpus") {
        Ok(rd) => rd.count() as u32,
        Err(_) => 0
    }
}

#[get("/healthz")]
async fn healthz() -> impl Responder { HttpResponse::Ok().finish() }

#[get("/metrics")]
async fn metrics() -> impl Responder {
    let node = std::env::var("MY_NODE_NAME").unwrap_or_else(|_| "unknown-node".to_string());
    let m = Metrics {
        node_name: node,
        cpu_usage_pct: read_cpu_usage_pct(),
        mem_usage_pct: read_mem_usage_pct(),
        gpu_count: read_gpu_count(),
    };
    HttpResponse::Ok().json(m)
}

#[actix_web::main]
async fn main() -> std::io::Result<()> {
    let filter = EnvFilter::try_from_default_env().unwrap_or_else(|_| EnvFilter::new("info"));
    tracing_subscriber::fmt().with_env_filter(filter).with_max_level(Level::INFO).init();
    info!("✅ Hyperion Agent (Phase 1 Final) listening on :9090");

    // This is the correct, long-running web server that will not exit.
    HttpServer::new(|| App::new().service(healthz).service(metrics))
        .bind(("0.0.0.0", 9090))?
        .run()
        .await
}