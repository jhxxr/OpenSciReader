from __future__ import annotations

import asyncio
import importlib
import inspect
import json
import os
import pathlib
import re
import sys
import time
import traceback
import types
import typing


OPENAI_COMPATIBLE_USER_AGENT = (
    "Mozilla/5.0 (Windows NT 10.0; Win64; x64) "
    "AppleWebKit/537.36 (KHTML, like Gecko) "
    "Chrome/123.0.0.0 Safari/537.36"
)


def emit(payload: dict) -> None:
    sys.stdout.write(json.dumps(payload, ensure_ascii=False) + "\n")
    sys.stdout.flush()


def install_openai_compatible_user_agent_patch() -> None:
    try:
        import openai
    except Exception:
        return

    if getattr(openai.OpenAI, "_openscireader_user_agent_patched", False):
        return

    original_init = openai.OpenAI.__init__

    def patched_init(self, *args, **kwargs):
        headers = dict(kwargs.get("default_headers") or {})
        headers.setdefault("User-Agent", OPENAI_COMPATIBLE_USER_AGENT)
        kwargs["default_headers"] = headers
        return original_init(self, *args, **kwargs)

    openai.OpenAI.__init__ = patched_init
    openai.OpenAI._openscireader_user_agent_patched = True


def summarize_exception_message(exc: BaseException) -> str:
    text = str(exc or "").replace("\r\n", "\n").strip()
    if not text:
        return exc.__class__.__name__

    for marker in (
        "\nTraceback (most recent call last):",
        "\nSubprocess traceback:",
        "\nReceived error from subprocess:",
    ):
        idx = text.find(marker)
        if idx >= 0:
            text = text[:idx].strip()

    lines = []
    for raw_line in text.splitlines():
        line = raw_line.strip()
        if not line:
            continue
        if "settings.basic.input_files is for cli" in line:
            line = line.split("settings.basic.input_files is for cli", 1)[0].rstrip(" :")
        line = re.sub(r"\s+", " ", line).strip()
        if not line:
            continue
        if line not in lines:
            lines.append(line)
        if len(lines) >= 3:
            break

    if not lines:
        return exc.__class__.__name__
    return " ".join(lines)


def read_request() -> dict:
    raw = sys.stdin.read().strip()
    if not raw:
        raise RuntimeError("worker request is empty")
    data = json.loads(raw)
    if not isinstance(data, dict):
        raise RuntimeError("worker request must be a JSON object")
    return data


def resolve_pdf2zh_runtime_dir() -> pathlib.Path | None:
    env_path = os.environ.get("OPENSCIREADER_PDF2ZH_RUNTIME_DIR", "").strip()
    candidates: list[pathlib.Path] = []
    if env_path:
        candidates.append(pathlib.Path(env_path))

    executable = pathlib.Path(sys.executable).resolve()
    candidates.extend(
        [
            executable.parent.parent,
            executable.parent,
        ]
    )

    for candidate in candidates:
        if not candidate:
            continue
        if (candidate / "site-packages").is_dir():
            return candidate
    return None


def restore_bundled_offline_assets() -> None:
    runtime_dir = resolve_pdf2zh_runtime_dir()
    if runtime_dir is None:
        return

    archives = sorted(runtime_dir.glob("offline_assets_*.zip"))
    if not archives:
        return

    from babeldoc.assets.assets import restore_offline_assets_package

    restore_offline_assets_package(archives[0])


def get_field(value, *names, default=None):
    for name in names:
        if isinstance(value, dict) and name in value:
            return value[name]
        if hasattr(value, name):
            return getattr(value, name)
    return default


def normalize_scalar(value):
    if value is None:
        return None
    if isinstance(value, (str, int, float, bool)):
        return value
    if isinstance(value, os.PathLike):
        return str(value)
    return str(value)


def normalize_stage_items(items) -> list[dict]:
    normalized = []
    for item in items or []:
        if item is None:
            continue
        name = normalize_scalar(
            get_field(item, "name", "stage", "title", default="")
        ) or ""
        percent = get_field(item, "percent", "progress", "value", default=0.0)
        try:
            percent = float(percent or 0.0)
        except (TypeError, ValueError):
            percent = 0.0
        normalized.append({"name": name, "percent": percent})
    return normalized


def merge_pdf_files(paths: list[str], output_path: pathlib.Path) -> str:
    import fitz

    normalized_paths = [str(pathlib.Path(item).expanduser().resolve()) for item in paths if item]
    if not normalized_paths:
        raise RuntimeError("no pdf files to merge")

    output_path.parent.mkdir(parents=True, exist_ok=True)
    merged_doc = fitz.open()
    try:
        for raw_path in normalized_paths:
            source_path = pathlib.Path(raw_path)
            if not source_path.is_file():
                raise FileNotFoundError(f"missing pdf chunk: {source_path}")
            source_doc = fitz.open(source_path)
            try:
                merged_doc.insert_pdf(source_doc)
            finally:
                source_doc.close()
        merged_doc.save(
            output_path,
            garbage=3,
            deflate=True,
            use_objstms=1,
        )
    finally:
        merged_doc.close()
    return str(output_path)


def interleave_pdf_files(
    original_pdf_path: str, translated_pdf_path: str, output_path: pathlib.Path
) -> str:
    import fitz

    original_path = pathlib.Path(original_pdf_path).expanduser().resolve()
    translated_path = pathlib.Path(translated_pdf_path).expanduser().resolve()
    if not original_path.is_file():
        raise FileNotFoundError(f"missing original pdf: {original_path}")
    if not translated_path.is_file():
        raise FileNotFoundError(f"missing translated pdf: {translated_path}")

    output_path.parent.mkdir(parents=True, exist_ok=True)
    original_doc = fitz.open(original_path)
    translated_doc = fitz.open(translated_path)
    merged_doc = fitz.open()
    try:
        total_pages = max(original_doc.page_count, translated_doc.page_count)
        for page_index in range(total_pages):
            if page_index < original_doc.page_count:
                merged_doc.insert_pdf(
                    original_doc, from_page=page_index, to_page=page_index
                )
            if page_index < translated_doc.page_count:
                merged_doc.insert_pdf(
                    translated_doc, from_page=page_index, to_page=page_index
                )
        merged_doc.save(
            output_path,
            garbage=3,
            deflate=True,
            use_objstms=1,
        )
    finally:
        merged_doc.close()
        translated_doc.close()
        original_doc.close()
    return str(output_path)


def normalize_output_stem(value, fallback: str = "translated-document") -> str:
    text = normalize_scalar(value) or ""
    text = re.sub(r"\s+", " ", text).strip()
    text = re.sub(r'[<>:"/\\\\|?*\x00-\x1f]+', "-", text)
    text = re.sub(r"[^0-9A-Za-z\.\-_ \u0080-\uffff]+", "-", text)
    text = re.sub(r"[- ]+", "-", text).strip(" .-_")
    return text or fallback


def normalized_path_text(value) -> str:
    text = normalize_scalar(value) or ""
    text = text.strip()
    if not text:
        return ""
    try:
        return str(pathlib.Path(text).expanduser().resolve())
    except Exception:
        return text


def preferred_pdf_output_path(result: dict, *names: str) -> str:
    for name in names:
        path = normalize_scalar(result.get(name)) or ""
        if str(path).strip():
            return str(path).strip()
    return ""


def move_pdf_output(source_path: str, target_path: pathlib.Path) -> str:
    source_text = normalized_path_text(source_path)
    if not source_text:
        return ""

    source = pathlib.Path(source_text)
    if not source.is_file():
        return source_path

    target = target_path.expanduser().resolve()
    target.parent.mkdir(parents=True, exist_ok=True)
    if source == target:
        return str(target)
    if target.exists():
        target.unlink()
    source.replace(target)
    return str(target)


def rewrite_output_aliases(result: dict, source_path: str, target_path: str, *names: str) -> None:
    source_key = normalized_path_text(source_path)
    target_key = normalized_path_text(target_path)
    if not source_key or not target_key:
        return
    for name in names:
        current = normalized_path_text(result.get(name))
        if current == source_key:
            result[name] = target_path


def finalize_export_translate_result(request: dict, result: dict) -> dict:
    finalized = dict(result or {})
    output_dir = pathlib.Path(request.get("outputDir", ".")).expanduser().resolve()
    output_dir.mkdir(parents=True, exist_ok=True)
    output_stem = normalize_output_stem(request.get("outputFileStem"))

    mono_source_path = preferred_pdf_output_path(
        finalized, "no_watermark_mono_pdf_path", "mono_pdf_path"
    )
    if mono_source_path:
        named_mono_path = move_pdf_output(
            mono_source_path, output_dir / f"{output_stem}.translated.pdf"
        )
        rewrite_output_aliases(
            finalized,
            mono_source_path,
            named_mono_path,
            "mono_pdf_path",
            "no_watermark_mono_pdf_path",
        )

    dual_source_path = preferred_pdf_output_path(
        finalized, "no_watermark_dual_pdf_path", "dual_pdf_path"
    )
    if dual_source_path:
        named_dual_path = move_pdf_output(
            dual_source_path, output_dir / f"{output_stem}.dual.pdf"
        )
        rewrite_output_aliases(
            finalized,
            dual_source_path,
            named_dual_path,
            "dual_pdf_path",
            "no_watermark_dual_pdf_path",
        )

    original_pdf_path = preferred_pdf_output_path(finalized, "original_pdf_path") or (
        normalize_scalar(request.get("inputPdfPath")) or ""
    )
    translated_pdf_path = preferred_pdf_output_path(
        finalized, "no_watermark_mono_pdf_path", "mono_pdf_path"
    )
    if original_pdf_path and translated_pdf_path:
        mixed_pdf_path = interleave_pdf_files(
            original_pdf_path,
            translated_pdf_path,
            output_dir / f"{output_stem}.interleaved.pdf",
        )
        finalized["mixed_pdf_path"] = mixed_pdf_path
        finalized["no_watermark_mixed_pdf_path"] = mixed_pdf_path

    if not finalized.get("original_pdf_path"):
        finalized["original_pdf_path"] = original_pdf_path
    return finalized


def normalize_translate_result(result) -> dict:
    if result is None:
        return {}
    return {
        "original_pdf_path": normalize_scalar(
            get_field(result, "original_pdf_path", "originalPdfPath")
        )
        or "",
        "mono_pdf_path": normalize_scalar(
            get_field(result, "mono_pdf_path", "monoPdfPath")
        )
        or "",
        "mixed_pdf_path": normalize_scalar(
            get_field(result, "mixed_pdf_path", "mixedPdfPath")
        )
        or "",
        "dual_pdf_path": normalize_scalar(
            get_field(result, "dual_pdf_path", "dualPdfPath")
        )
        or "",
        "no_watermark_mono_pdf_path": normalize_scalar(
            get_field(
                result,
                "no_watermark_mono_pdf_path",
                "noWatermarkMonoPdfPath",
            )
        )
        or "",
        "no_watermark_mixed_pdf_path": normalize_scalar(
            get_field(
                result,
                "no_watermark_mixed_pdf_path",
                "noWatermarkMixedPdfPath",
            )
        )
        or "",
        "no_watermark_dual_pdf_path": normalize_scalar(
            get_field(
                result,
                "no_watermark_dual_pdf_path",
                "noWatermarkDualPdfPath",
            )
        )
        or "",
        "auto_extracted_glossary_path": normalize_scalar(
            get_field(
                result,
                "auto_extracted_glossary_path",
                "autoExtractedGlossaryPath",
            )
        )
        or "",
        "total_seconds": float(
            get_field(result, "total_seconds", "totalSeconds", default=0.0) or 0.0
        ),
        "peak_memory_usage": float(
            get_field(
                result, "peak_memory_usage", "peakMemoryUsage", default=0.0
            )
            or 0.0
        ),
    }


def normalize_event(event) -> dict:
    event_type = normalize_scalar(get_field(event, "type", "event", default="")) or ""
    payload = {"type": event_type}

    if event_type == "stage_summary":
        payload["stages"] = normalize_stage_items(
            get_field(event, "stages", "summary", default=[])
        )
        payload["part_index"] = int(
            get_field(event, "part_index", "partIndex", default=0) or 0
        )
        payload["total_parts"] = int(
            get_field(event, "total_parts", "totalParts", default=0) or 0
        )
        return payload

    if event_type in {"progress_start", "progress_update", "progress_end"}:
        payload["stage"] = normalize_scalar(
            get_field(event, "stage", "name", default="")
        ) or ""
        payload["stage_progress"] = float(
            get_field(event, "stage_progress", "progress", default=0.0) or 0.0
        )
        payload["overall_progress"] = float(
            get_field(
                event, "overall_progress", "overall", "overallPercent", default=0.0
            )
            or 0.0
        )
        payload["stage_current"] = int(
            get_field(event, "stage_current", "current", default=0) or 0
        )
        payload["stage_total"] = int(
            get_field(event, "stage_total", "total", default=0) or 0
        )
        payload["part_index"] = int(
            get_field(event, "part_index", "partIndex", default=0) or 0
        )
        payload["total_parts"] = int(
            get_field(event, "total_parts", "totalParts", default=0) or 0
        )
        return payload

    if event_type == "finish":
        payload["translate_result"] = normalize_translate_result(
            get_field(event, "translate_result", "translateResult")
        )
        return payload

    if event_type == "error":
        payload["error"] = normalize_scalar(
            get_field(event, "error", "message", default="")
        ) or "unknown worker error"
        payload["error_type"] = normalize_scalar(
            get_field(event, "error_type", "errorType", default="")
        ) or ""
        payload["details"] = normalize_scalar(
            get_field(event, "details", "traceback", default="")
        ) or ""
        return payload

    payload["raw"] = normalize_scalar(event)
    return payload


def iter_model_fields(model_cls):
    return getattr(model_cls, "model_fields", None) or getattr(
        model_cls, "__fields__", {}
    )


def normalize_key(value: str) -> str:
    return (
        value.strip()
        .replace("_", "-")
        .replace(" ", "-")
        .replace(".", "-")
        .lower()
    )


def unwrap_annotation(annotation):
    if annotation is None:
        return None
    origin = typing.get_origin(annotation)
    if origin in {typing.Union, types.UnionType}:
        args = [item for item in typing.get_args(annotation) if item is not type(None)]
        if len(args) == 1:
            return unwrap_annotation(args[0])
    return annotation


def is_model_class(annotation) -> bool:
    annotation = unwrap_annotation(annotation)
    return isinstance(annotation, type) and bool(iter_model_fields(annotation))


def get_model_annotation(field, fallback):
    annotation = getattr(field, "annotation", None)
    if annotation is None:
        annotation = getattr(field, "outer_type_", None)
    if annotation is None:
        annotation = fallback
    return unwrap_annotation(annotation)


def lookup_flat_value(flat: dict, *names):
    normalized_names = {normalize_key(name) for name in names if name}
    for key, value in flat.items():
        if normalize_key(key) in normalized_names:
            return value, True
    return None, False


def build_nested_payload(model_cls, flat: dict) -> dict:
    payload = {}
    for field_name, field in iter_model_fields(model_cls).items():
        alias = getattr(field, "alias", None) or field_name
        annotation = get_model_annotation(field, None)
        if is_model_class(annotation):
            nested = build_nested_payload(annotation, flat)
            if nested:
                payload[field_name] = nested
            continue
        value, found = lookup_flat_value(flat, field_name, alias)
        if found:
            payload[field_name] = value
    return payload


def toml_value(value):
    if isinstance(value, bool):
        return "true" if value else "false"
    if isinstance(value, int):
        return str(value)
    if isinstance(value, float):
        return repr(value)
    text = str(value).replace("\\", "\\\\").replace('"', '\\"')
    return f'"{text}"'


def write_toml_config(flat: dict, config_path: pathlib.Path) -> pathlib.Path:
    lines = ["[babeldoc]"]
    for key, value in flat.items():
        if value in (None, ""):
            continue
        lines.append(f"{key} = {toml_value(value)}")
    config_path.write_text("\n".join(lines) + "\n", encoding="utf-8")
    return config_path


def try_call(func, *positional, **keyword):
    try:
        signature = inspect.signature(func)
    except (TypeError, ValueError):
        return func(*positional, **keyword)

    bound_kwargs = {}
    positional_values = list(positional)
    for name, parameter in signature.parameters.items():
        if positional_values and parameter.kind in (
            inspect.Parameter.POSITIONAL_ONLY,
            inspect.Parameter.POSITIONAL_OR_KEYWORD,
        ):
            bound_kwargs[name] = positional_values.pop(0)
            continue
        if name in keyword:
            bound_kwargs[name] = keyword[name]
    if positional_values:
        return func(*positional, **keyword)
    return func(**bound_kwargs)


def find_settings_loader():
    module_names = [
        "pdf2zh_next.config",
        "pdf2zh_next.config.loader",
        "pdf2zh_next.config.settings",
        "pdf2zh_next.settings",
    ]
    function_names = [
        "load_settings",
        "load_settings_from_file",
        "load_settings_from_toml",
        "read_settings",
        "parse_settings_file",
    ]
    for module_name in module_names:
        try:
            module = importlib.import_module(module_name)
        except Exception:
            continue
        for function_name in function_names:
            function = getattr(module, function_name, None)
            if callable(function):
                return function
    return None


def find_settings_model():
    module_names = [
        "pdf2zh_next.config",
        "pdf2zh_next.config.settings",
        "pdf2zh_next.config.model",
        "pdf2zh_next.settings",
    ]
    for module_name in module_names:
        try:
            module = importlib.import_module(module_name)
        except Exception:
            continue
        for attr_name in ("SettingsModel", "Settings", "BabelDocSettings"):
            model_cls = getattr(module, attr_name, None)
            if isinstance(model_cls, type):
                return model_cls
    return None


def build_settings(request: dict):
    provider = request.get("provider") or {}
    provider_api_key = provider.get("apiKey", "")

    structured = {
        "report_interval": float(request.get("reportInterval", 0.25) or 0.25),
        "basic": {
            "input_files": [request.get("inputPdfPath", "")],
            "debug": False,
        },
        "translation": {
            "output": request.get("outputDir", ""),
            "lang_in": request.get("sourceLang", "en"),
            "lang_out": request.get("targetLang", "zh-CN"),
            "qps": int(request.get("qps", 4) or 4),
            "pool_max_workers": int(request.get("poolMaxWorkers", 4) or 4),
            "term_pool_max_workers": int(
                request.get("termPoolMaxWorkers", 0) or 0
            )
            or None,
            "min_text_length": int(request.get("minTextLength", 5) or 5),
        },
        "pdf": {
            "pages": request.get("pages", "") or None,
            "only_include_translated_page": bool(
                request.get("onlyIncludeTranslatedPage", False)
            ),
            "no_dual": bool(request.get("noDual", False)),
            "no_mono": bool(request.get("noMono", False)),
            "max_pages_per_part": int(request.get("maxPagesPerPart", 0) or 0) or None,
            "watermark_output_mode": request.get(
                "watermarkOutputMode", "no_watermark"
            ),
        },
        "translate_engine_settings": {
            "translate_engine_type": "OpenAICompatible",
            "openai_compatible_model": provider.get("modelId", ""),
            "openai_compatible_base_url": provider.get("baseUrl", ""),
            "openai_compatible_api_key": provider_api_key,
        },
    }

    flat = {
        "debug": False,
        "working-dir": request.get("workingDir", ""),
        "output": request.get("outputDir", ""),
        "lang-in": request.get("sourceLang", "en"),
        "lang-out": request.get("targetLang", "zh-CN"),
        "pages": request.get("pages", ""),
        "only-include-translated-page": bool(
            request.get("onlyIncludeTranslatedPage", False)
        ),
        "no-dual": bool(request.get("noDual", False)),
        "no-mono": bool(request.get("noMono", False)),
        "report-interval": float(request.get("reportInterval", 0.25) or 0.25),
        "max-pages-per-part": int(request.get("maxPagesPerPart", 0) or 0),
        "qps": int(request.get("qps", 4) or 4),
        "pool-max-workers": int(request.get("poolMaxWorkers", 4) or 4),
        "term-pool-max-workers": int(request.get("termPoolMaxWorkers", 0) or 0),
        "min-text-length": int(request.get("minTextLength", 5) or 5),
        "watermark-output-mode": request.get("watermarkOutputMode", "no_watermark"),
        "openai-compatible-model": provider.get("modelId", ""),
        "openai-compatible-base-url": provider.get("baseUrl", ""),
        "openai-compatible-api-key": provider_api_key,
    }

    config_path = pathlib.Path(request.get("workingDir", ".")) / "babeldoc.worker.toml"
    write_toml_config(flat, config_path)

    loader = find_settings_loader()
    if callable(loader):
        for argument in (str(config_path), config_path):
            try:
                return try_call(
                    loader,
                    argument,
                    config_path=str(config_path),
                    file_path=str(config_path),
                    path=str(config_path),
                )
            except Exception:
                continue

    settings_model = find_settings_model()
    if settings_model is None:
        raise RuntimeError("unable to locate pdf2zh_next SettingsModel")

    candidates = [
        structured,
        build_nested_payload(settings_model, flat),
        flat,
        {"babeldoc": flat},
    ]
    validators = [
        getattr(settings_model, "model_validate", None),
        getattr(settings_model, "parse_obj", None),
        getattr(settings_model, "validate", None),
    ]
    validators = [item for item in validators if callable(item)]
    for validator in validators:
        for candidate in candidates:
            if not candidate:
                continue
            try:
                return validator(candidate)
            except Exception:
                continue

    for candidate in candidates:
        if not candidate:
            continue
        try:
            return settings_model(**candidate)
        except Exception:
            continue

    raise RuntimeError("unable to build pdf2zh_next settings from request")


async def run_worker(request: dict) -> int:
    if request.get("mergeMonoPdfPaths") or request.get("mergeDualPdfPaths"):
        return await run_merge_worker(request)

    try:
        restore_bundled_offline_assets()
    except BaseException as exc:
        emit(
            {
                "type": "error",
                "error": f"restore offline assets failed: {exc}",
                "error_type": exc.__class__.__name__,
                "details": traceback.format_exc(),
            }
        )
        return 1

    try:
        install_openai_compatible_user_agent_patch()
        module = importlib.import_module("pdf2zh_next.high_level")
        do_translate_async_stream = getattr(module, "do_translate_async_stream")
    except Exception as exc:
        emit(
            {
                "type": "error",
                "error": f"import pdf2zh_next.high_level failed: {summarize_exception_message(exc)}",
                "error_type": exc.__class__.__name__,
                "details": traceback.format_exc(),
            }
        )
        return 1

    input_path = pathlib.Path(request["inputPdfPath"]).expanduser().resolve()
    pathlib.Path(request["workingDir"]).mkdir(parents=True, exist_ok=True)
    pathlib.Path(request["outputDir"]).mkdir(parents=True, exist_ok=True)

    try:
        settings = build_settings(request)
        # We pass the target PDF through do_translate_async_stream(file=...).
        # Keeping basic.input_files set only triggers a warning in pdf2zh_next.
        if getattr(getattr(settings, "basic", None), "input_files", None):
            settings.basic.input_files = set()
        # OpenSciReader already isolates translation in this dedicated worker process.
        # Running pdf2zh_next in-process avoids a nested Python subprocess on Windows,
        # where the embeddable runtime ignores PYTHONPATH/sitecustomize and loses our
        # OpenAI-compatible user-agent patch.
        settings.basic.debug = False
    except Exception as exc:
        emit(
            {
                "type": "error",
                "error": f"build settings failed: {summarize_exception_message(exc)}",
                "error_type": exc.__class__.__name__,
                "details": traceback.format_exc(),
            }
        )
        return 1

    try:
        async for event in do_translate_async_stream(settings=settings, file=str(input_path)):
            payload = normalize_event(event)
            if payload.get("type") == "finish" and request.get("mode") == "export":
                emit(
                    {
                        "type": "progress_start",
                        "stage": "prepare export files",
                        "stage_progress": 0.0,
                        "overall_progress": 0.97,
                        "stage_current": 0,
                        "stage_total": 1,
                    }
                )
                try:
                    payload["translate_result"] = finalize_export_translate_result(
                        request, payload.get("translate_result") or {}
                    )
                except Exception as exc:
                    emit(
                        {
                            "type": "error",
                            "error": f"prepare export files failed: {summarize_exception_message(exc)}",
                            "error_type": exc.__class__.__name__,
                            "details": traceback.format_exc(),
                        }
                    )
                    return 1
                emit(
                    {
                        "type": "progress_end",
                        "stage": "prepare export files",
                        "stage_progress": 1.0,
                        "overall_progress": 1.0,
                        "stage_current": 1,
                        "stage_total": 1,
                    }
                )
            emit(payload)
    except Exception as exc:
        emit(
            {
                "type": "error",
                "error": summarize_exception_message(exc),
                "error_type": exc.__class__.__name__,
                "details": traceback.format_exc(),
            }
        )
        return 1
    return 0


async def run_merge_worker(request: dict) -> int:
    mono_paths = request.get("mergeMonoPdfPaths") or []
    dual_paths = request.get("mergeDualPdfPaths") or []
    output_dir = pathlib.Path(request.get("outputDir", ".")).expanduser().resolve()
    output_dir.mkdir(parents=True, exist_ok=True)
    output_stem = normalize_output_stem(request.get("outputFileStem"))
    should_build_mixed = bool(mono_paths) and bool(request.get("inputPdfPath"))
    stage_total = max(
        int(bool(mono_paths)) + int(should_build_mixed) + int(bool(dual_paths)), 1
    )

    emit(
        {
            "type": "progress_start",
            "stage": "merge preview outputs",
            "stage_progress": 0.0,
            "overall_progress": 0.05,
            "stage_current": 0,
            "stage_total": stage_total,
        }
    )

    started = time.perf_counter()
    try:
        mono_output_path = ""
        mixed_output_path = ""
        dual_output_path = ""
        stage_current = 0
        if mono_paths:
            mono_output_path = merge_pdf_files(
                mono_paths, output_dir / f"{output_stem}.translated.pdf"
            )
            stage_current += 1
            emit(
                {
                    "type": "progress_update",
                    "stage": "merge preview outputs",
                    "stage_progress": stage_current / stage_total,
                    "overall_progress": 0.35,
                    "stage_current": stage_current,
                    "stage_total": stage_total,
                }
            )
        if should_build_mixed and mono_output_path:
            mixed_output_path = interleave_pdf_files(
                request.get("inputPdfPath", ""),
                mono_output_path,
                output_dir / f"{output_stem}.interleaved.pdf",
            )
            stage_current += 1
            emit(
                {
                    "type": "progress_update",
                    "stage": "merge preview outputs",
                    "stage_progress": stage_current / stage_total,
                    "overall_progress": 0.7 if dual_paths else 0.95,
                    "stage_current": stage_current,
                    "stage_total": stage_total,
                }
            )
        if dual_paths:
            dual_output_path = merge_pdf_files(
                dual_paths, output_dir / f"{output_stem}.dual.pdf"
            )
            stage_current += 1
            emit(
                {
                    "type": "progress_update",
                    "stage": "merge preview outputs",
                    "stage_progress": stage_current / stage_total,
                    "overall_progress": 0.95,
                    "stage_current": stage_current,
                    "stage_total": stage_total,
                }
            )
    except Exception as exc:
        emit(
            {
                "type": "error",
                "error": str(exc),
                "error_type": exc.__class__.__name__,
                "details": traceback.format_exc(),
            }
        )
        return 1

    emit(
        {
            "type": "progress_end",
            "stage": "merge preview outputs",
            "stage_progress": 1.0,
            "overall_progress": 1.0,
            "stage_current": stage_total,
            "stage_total": stage_total,
        }
    )
    emit(
        {
            "type": "finish",
            "translate_result": {
                "original_pdf_path": request.get("inputPdfPath", "") or "",
                "mono_pdf_path": mono_output_path,
                "mixed_pdf_path": mixed_output_path,
                "dual_pdf_path": dual_output_path,
                "no_watermark_mono_pdf_path": mono_output_path,
                "no_watermark_mixed_pdf_path": mixed_output_path,
                "no_watermark_dual_pdf_path": dual_output_path,
                "auto_extracted_glossary_path": "",
                "total_seconds": time.perf_counter() - started,
                "peak_memory_usage": 0.0,
            },
        }
    )
    return 0


def main() -> int:
    try:
        request = read_request()
    except Exception as exc:
        emit(
            {
                "type": "error",
                "error": str(exc),
                "error_type": exc.__class__.__name__,
                "details": traceback.format_exc(),
            }
        )
        return 1

    return asyncio.run(run_worker(request))


if __name__ == "__main__":
    raise SystemExit(main())
