from workers.common.llm.config import _resolve_disable_thinking


def test_resolve_disable_thinking_prefers_worker_disable(monkeypatch):
    monkeypatch.setenv("SOURCEBRIDGE_WORKER_LLM_DISABLE_THINKING", "true")
    monkeypatch.setenv("SOURCEBRIDGE_LLM_ENABLE_THINKING", "true")

    assert _resolve_disable_thinking() is True


def test_resolve_disable_thinking_report_scope_can_override(monkeypatch):
    monkeypatch.delenv("SOURCEBRIDGE_WORKER_LLM_DISABLE_THINKING", raising=False)
    monkeypatch.delenv("SOURCEBRIDGE_WORKER_LLM_ENABLE_THINKING", raising=False)
    monkeypatch.setenv("SOURCEBRIDGE_WORKER_LLM_REPORT_ENABLE_THINKING", "false")

    assert _resolve_disable_thinking(report=True) is True


def test_resolve_disable_thinking_global_enable_disables_flag(monkeypatch):
    monkeypatch.delenv("SOURCEBRIDGE_WORKER_LLM_DISABLE_THINKING", raising=False)
    monkeypatch.delenv("SOURCEBRIDGE_WORKER_LLM_ENABLE_THINKING", raising=False)
    monkeypatch.delenv("SOURCEBRIDGE_WORKER_LLM_REPORT_ENABLE_THINKING", raising=False)
    monkeypatch.setenv("SOURCEBRIDGE_LLM_ENABLE_THINKING", "true")

    assert _resolve_disable_thinking() is False
