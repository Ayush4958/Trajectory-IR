from __future__ import annotations

import zipfile
from pathlib import Path

ROOT = Path(__file__).resolve().parents[2]


def _built_wheel() -> Path:
    wheels = sorted((ROOT / "dist").glob("trajectory_ir-*.whl"))
    if len(wheels) != 1:
        found = ", ".join(path.name for path in wheels) or "none"
        raise AssertionError(
            f"expected one trajectory_ir wheel in dist/, found {found}. Run python -m build first."
        )
    return wheels[0]


def assert_wheel_nests_client_and_drivers(wheel: Path) -> None:
    with zipfile.ZipFile(wheel) as zf:
        names = zf.namelist()
    tops = {name.split("/", 1)[0] for name in names}
    assert "client" not in tops
    assert "drivers" not in tops
    assert any(name.startswith("trajectory_ir/client/") for name in names)
    assert any(name.startswith("trajectory_ir/drivers/") for name in names)


def test_wheel_ships_client_and_drivers_under_trajectory_ir() -> None:
    assert_wheel_nests_client_and_drivers(_built_wheel())


def main() -> None:
    assert_wheel_nests_client_and_drivers(_built_wheel())


if __name__ == "__main__":
    main()
