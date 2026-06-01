import time
import subprocess
import mss
from PIL import Image, ImageOps
import os
import pyperclip

# ========================= CONFIGURATION =========================
SAVE_FOLDER = "screenshots"
INTERVAL_SECONDS = 0.0
TYPE_DELAY = 0.02
# ================================================================

os.makedirs(SAVE_FOLDER, exist_ok=True)

selected_region = None


def select_region_with_slop():
    global selected_region

    print("Click and drag to select capture region...")

    try:
        result = subprocess.check_output(
            ['slop', '-f', '%x %y %w %h']
        ).decode().strip()

        x, y, w, h = map(int, result.split())

        if w < 16 or h < 16:
            print("Region too small.")
            return False

        selected_region = {
            "left": x,
            "top": y,
            "width": w,
            "height": h
        }

        print(f"Selected region: {selected_region}")
        return True

    except Exception as e:
        print(f"Selection failed: {e}")
        return False


def image_to_digit_string(img):
    pixels = list(img.getdata())

    # 0-9 mapping
    return ''.join(str(min(p // 26, 9)) for p in pixels)


def focus_delay():
    print("\nFocus your target window...")
    for i in range(3, 0, -1):
        print(f"Starting in {i}...")
        time.sleep(1)


def paste_text_xorg(text):
    """
    Reliable X11 clipboard paste using xclip + xdotool
    """

    # Copy to clipboard
    pyperclip.copy(text)

    # Small delay for clipboard ownership
    time.sleep(0.1)

    # Ctrl+V
    subprocess.run(
        ['xdotool', 'key', '--clearmodifiers', 'ctrl+v'],
        check=True
    )


def press_enter():
    subprocess.run(
        ['xdotool', 'key', '--clearmodifiers', 'Return'],
        check=True
    )


def type_zero():
    subprocess.run(
        ['xdotool', 'type', '--delay', str(int(TYPE_DELAY * 1000)), '0'],
        check=True
    )


def save_debug_image(img):
    timestamp = time.strftime("%Y%m%d_%H%M%S")
    path = f"{SAVE_FOLDER}/capture_{timestamp}.png"
    img.save(path)


def main_loop():

    if not selected_region:
        return

    focus_delay()

    print("\nRunning... Press Ctrl+C to stop.\n")

    with mss.mss() as sct:

        try:
            while True:

                # Capture region
                screenshot = sct.grab(selected_region)

                img = Image.frombytes(
                    "RGB",
                    screenshot.size,
                    screenshot.rgb
                )

                # Downscale
                img = img.resize((16, 16), Image.LANCZOS)

                # Grayscale
                img = ImageOps.grayscale(img)

                # Convert to digit string
                digit_string = image_to_digit_string(img)

                # Save debug image
                save_debug_image(img)

                # Send directly without clipboard
                subprocess.run([
                    'xdotool',
                    'type',
                    '--delay',
                    '1',
                    digit_string
                ], check=True)

                time.sleep(0.05)

                subprocess.run([
                    'xdotool',
                    'key',
                    'Return'
                ], check=True)

                print(f"Sent {len(digit_string)} digits")

                time.sleep(INTERVAL_SECONDS)

        except KeyboardInterrupt:
            print("\nStopped.")

        except Exception as e:
            print(f"Runtime error: {e}")


if __name__ == "__main__":

    print("=== Xorg Screen → Digits Automator ===")

    if select_region_with_slop():
        main_loop()