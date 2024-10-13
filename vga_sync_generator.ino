// compatible with Arduino Nano + Mega
// courtesy of https://scanlines.xyz/t/tutorials-for-generating-video-sync-signals-with-arduino/104/4

#define LINE_CYCLES 508
#define HSYNC_CYCLES 60
#define VSYNC_LINES 2
#define FRAME_LINES 525

#define VSYNC_HIGH bitWrite(PORTD, 7, 1)
#define VSYNC_LOW bitWrite(PORTD, 7, 0)

volatile int linecount;

void setup() {
    pinMode(7, OUTPUT); // VSync
    pinMode(9, OUTPUT); // HSync
    // inverted fast pwm mode on timer 1
    TCCR1A = _BV(COM1A1) | _BV(COM1A0) | _BV(WGM11);
    TCCR1B = _BV(WGM13) | _BV(WGM12) | _BV(CS10);

    ICR1 = LINE_CYCLES; // overflow at cycles per line
    OCR1A = HSYNC_CYCLES; // compare high after HSync cycles

    TIMSK1 = _BV(TOIE1); // enable timer overflow interrupt
}

ISR(TIMER1_OVF_vect) {
    switch(linecount) {
        case 0:
            VSYNC_LOW;
            linecount++;
        break;
        case 2:
            VSYNC_HIGH;
            linecount++;
        break;
        case FRAME_LINES:
            linecount = 0;
        break;
        default:
            linecount++;
    }
}

void loop() {}
