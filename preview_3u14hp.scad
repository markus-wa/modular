$fn=16;

// Dimensions
panel_width = floor(14 * 5.08);    // 14HP
panel_height = 128.5;  // 3U
panel_thickness = 2.00;

// Mounting holes
mount_offset_y = 3;
mount_offset_x = 7.45;
mount_offset2_x = 7.45 + (11 * 5.08);
mounting_hole_diameter = 3.2; // M3

// Compoonent hole sizes
jack_hole_diameter = 6; // 3.5mm

jack_height = 10;
lcd_height = 49;
lcd_width = 67;
screen_height = 42;
screen_width = 56.2;

module mounting_hole(x_pos, y_pos) {
    translate([x_pos, y_pos, panel_thickness / 2])
        cylinder(d = mounting_hole_diameter, h = panel_thickness + 1, center = true);
}

module jack(x_pos, y_pos) {
    translate([x_pos, y_pos, panel_thickness / 2])
        cylinder(d = jack_hole_diameter, h = panel_thickness + 1, center = true);
}

module lcd(y_pos) {
    translate([(panel_width - screen_width)/2, y_pos, -1])
        cube([screen_width, screen_height, panel_thickness + 2]);
}


buffer = 0.5;

jack2lcd_y = jack_height / 2 + buffer;
lcd2lcd_y = lcd_height + buffer;

jacks_y = 15;
lcd1_y = jacks_y + jack2lcd_y;
lcd2_y = lcd1_y + lcd2lcd_y;

module panel() {
    difference() {
        // Base panel
        cube([panel_width, panel_height, panel_thickness]);

        // Mounting holes
        mounting_hole(mount_offset2_x, mount_offset_y);
        mounting_hole(mount_offset2_x, panel_height - mount_offset_y);
        mounting_hole(mount_offset_x, panel_height - mount_offset_y);
        mounting_hole(mount_offset_x, mount_offset_y);

        // Components (Jacks, Toggles, Pot)
        jack(panel_width / 3, jacks_y);
        jack(panel_width - (panel_width / 3), jacks_y);
        lcd(lcd1_y);
        lcd(lcd2_y);
    }
}

panel();
