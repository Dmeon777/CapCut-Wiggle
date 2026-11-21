# Установщик CapCut Wiggle

Один `.exe` на Go без сторонних библиотек: своё окно в стиле Wiggle, установка в
`%LOCALAPPDATA%\Programs\CapCut Wiggle` без прав администратора, ярлыки на рабочем столе и в «Пуск»,
запись в «Приложения» Windows и удаление оттуда же.

Что делает при установке:
1. Скачивает свежую программу из последнего релиза
   `https://github.com/Dmeon777/CapCut-Wiggle/releases/latest/download/capcut-wiggle.zip`.
   Нет интернета или файла — ставит копию, вшитую в установщик (`app.zip`).
2. Скачивает встроенный Python 3.12 с python.org (если его ещё нет в папке программы).
3. Распаковывает, создаёт ярлыки, кладёт `uninstall.exe` и пишет программу в «Приложения».

Важно для релизов: файл в релизе должен называться `capcut-wiggle.zip`, внутри — `app.py` и остальные файлы
программы (можно в подпапке).

Сборка: `build.sh` (Linux/WSL) или `build.bat` (Windows, нужен Go). Новая вшитая копия — замени `app.zip`.

| Файл | Что это |
|---|---|
| `main_windows.go` | Точка входа: установка или `/uninstall` |
| `ui_windows.go`, `layout.go`, `draw.go`, `text_windows.go` | Окно, экраны, отрисовка |
| `install_windows.go`, `archive.go`, `shell_windows.go` | Установка, удаление, ярлыки, реестр |
| `win_windows.go` | Вызовы Win32 |
| `mksyso.py`, `setup.manifest`, `rsrc_windows_amd64.syso` | Иконка и манифест внутри `.exe` |
