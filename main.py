"""
QR Scanner — сканирует QR через веб-камеру (зеркально, как в зеркале),
"открывает" ссылку в фоне (HTTP GET-запрос, без видимого браузера),
пищит при успешном скане и показывает статус на весь экран.

Установка зависимостей:
    pip install opencv-python requests --break-system-packages
(numpy и simpleaudio больше не нужны — звук генерируется через стандартный
модуль wave и играется системным aplay/paplay, которые есть в Fedora из коробки)

Запуск:
    python3 qr_scanner.py

Управление:
    q — выход
    f — переключить полноэкранный режим
"""

import cv2
import requests
import time
import math
import wave
import struct
import subprocess
import tempfile
import shutil
import os

COOLDOWN_SECONDS = 3  # чтобы один и тот же QR не триггерился каждый кадр
REQUEST_TIMEOUT = 5
WINDOW_NAME = "QR Scanner"

SAMPLE_RATE = 44100

# ищем доступный проигрыватель один раз при старте
_PLAYER = shutil.which("paplay") or shutil.which("aplay") or shutil.which("ffplay")


def make_beep_wav(path, frequency=1000, duration_ms=150, volume=0.5):
    """Генерирует короткий wav-файл со звуковым сигналом без внешних библиотек."""
    n_samples = int(SAMPLE_RATE * duration_ms / 1000)
    with wave.open(path, "w") as wav_file:
        wav_file.setnchannels(1)
        wav_file.setsampwidth(2)  # 16 бит
        wav_file.setframerate(SAMPLE_RATE)
        frames = bytearray()
        for i in range(n_samples):
            t = i / SAMPLE_RATE
            value = int(volume * 32767 * math.sin(2 * math.pi * frequency * t))
            frames += struct.pack("<h", value)
        wav_file.writeframes(bytes(frames))


_TMP_DIR = tempfile.mkdtemp(prefix="qr_scanner_")
BEEP_OK_PATH = os.path.join(_TMP_DIR, "beep_ok.wav")
BEEP_ERR_PATH = os.path.join(_TMP_DIR, "beep_err.wav")
make_beep_wav(BEEP_OK_PATH, frequency=1200, duration_ms=150)
make_beep_wav(BEEP_ERR_PATH, frequency=400, duration_ms=250)


def play_beep(success: bool):
    if _PLAYER is None:
        print("[звук] Не найден aplay/paplay/ffplay — звук пропущен.")
        return
    path = BEEP_OK_PATH if success else BEEP_ERR_PATH
    try:
        # ffplay нужен флаг для тихого запуска без окна и без автозакрытия
        if _PLAYER.endswith("ffplay"):
            cmd = [_PLAYER, "-nodisp", "-autoexit", "-loglevel", "quiet", path]
        else:
            cmd = [_PLAYER, path]
        subprocess.Popen(cmd, stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL)
    except Exception as e:
        print(f"Не удалось воспроизвести звук: {e}")


def open_in_background(url: str) -> bool:
    """Делает GET-запрос по ссылке из QR, не открывая браузер."""
    try:
        resp = requests.get(url, timeout=REQUEST_TIMEOUT, allow_redirects=True)
        print(f"[OK] {url} -> статус {resp.status_code}")
        return resp.ok
    except requests.RequestException as e:
        print(f"[ERROR] Не удалось открыть {url}: {e}")
        return False


def main():
    cap = cv2.VideoCapture(0)
    if not cap.isOpened():
        print("Не удалось открыть камеру. Проверь индекс устройства (0, 1, ...).")
        return

    detector = cv2.QRCodeDetector()

    last_data = None
    last_scan_time = 0.0
    status_text = ""
    status_until = 0.0
    fullscreen = True

    cv2.namedWindow(WINDOW_NAME, cv2.WINDOW_NORMAL)
    cv2.setWindowProperty(WINDOW_NAME, cv2.WND_PROP_FULLSCREEN, cv2.WINDOW_FULLSCREEN)

    print("Наведи камеру на QR-код. 'q' — выход, 'f' — переключить полный экран.")
    print("Важно: клавиши работают, только когда окно с видео в фокусе — кликни по нему мышкой.")
    print("Если 'q' не реагирует — жми Ctrl+C в терминале, программа теперь закроется чисто.")

    while True:
        ret, frame = cap.read()
        if not ret:
            print("Кадр не получен, останавливаюсь.")
            break

        # зеркальное отражение — как будто смотришь в зеркало
        frame = cv2.flip(frame, 1)

        try:
            data, points, _ = detector.detectAndDecode(frame)
        except cv2.error:
            # OpenCV иногда находит "контур" нулевой площади (блик, шум) и падает
            # на decode() — просто считаем, что в этом кадре QR не найден
            data, points = "", None

        now = time.time()
        if data and (data != last_data or now - last_scan_time > COOLDOWN_SECONDS):
            last_data = data
            last_scan_time = now
            print(f"Найден QR: {data}")

            success = open_in_background(data)
            play_beep(success)
            status_text = "Done" if success else "Error"
            status_until = now + 2

        # рамка вокруг найденного QR (координаты points уже соответствуют
        # отражённому кадру, т.к. flip был до detectAndDecode)
        if points is not None:
            pts = points.astype(int).reshape(-1, 2)
            for i in range(len(pts)):
                cv2.line(frame, tuple(pts[i]), tuple(pts[(i + 1) % len(pts)]), (0, 255, 0), 2)

        # статус крупно по центру экрана
        if time.time() < status_until:
            color = (0, 255, 0) if status_text == "Done" else (0, 0, 255)
            h, w = frame.shape[:2]
            text_size = cv2.getTextSize(status_text, cv2.FONT_HERSHEY_SIMPLEX, 2.5, 5)[0]
            tx = (w - text_size[0]) // 2
            ty = (h + text_size[1]) // 2
            cv2.putText(frame, status_text, (tx, ty), cv2.FONT_HERSHEY_SIMPLEX, 2.5, color, 5)

        cv2.imshow(WINDOW_NAME, frame)

        key = cv2.waitKey(1) & 0xFF
        if key == ord('q'):
            break
        elif key == ord('f'):
            fullscreen = not fullscreen
            prop = cv2.WINDOW_FULLSCREEN if fullscreen else cv2.WINDOW_NORMAL
            cv2.setWindowProperty(WINDOW_NAME, cv2.WND_PROP_FULLSCREEN, prop)

    cap.release()
    cv2.destroyAllWindows()


if __name__ == "__main__":
    try:
        main()
    except KeyboardInterrupt:
        print("\nОстановлено пользователем (Ctrl+C). Закрываю камеру и окно...")
        cv2.destroyAllWindows()