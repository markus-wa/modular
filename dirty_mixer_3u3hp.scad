// Dimensions
panel_width = 15.00;    // 3HP
panel_height = 128.50;  // 3U
panel_thickness = 2.00;

// Mounting holes
mount_offset_y = 3;
mount_offset_x = 4;
mounting_hole_diameter = 3.2; // M3

// Compoonent hole sizes
jack_hole_diameter = 6; // 3.5mm
toggle_hole_diameter = 7;
pot_hole_diameter = 8;

module jack(y_pos) {
    translate([panel_width / 2, y_pos, panel_thickness / 2])
        cylinder(d = jack_hole_diameter, h = panel_thickness + 1, center = true);
}

module toggle(y_pos) {
    translate([panel_width / 2, y_pos, panel_thickness / 2])
        cylinder(d = toggle_hole_diameter, h = panel_thickness + 1, center = true);
}

module pot(y_pos) {
    translate([panel_width / 2, y_pos, panel_thickness / 2])
        cylinder(d = pot_hole_diameter, h = panel_thickness + 1, center = true);
}

module mounting_hole(x_pos, y_pos) {
    translate([x_pos, y_pos, panel_thickness / 2])
        cylinder(d = mounting_hole_diameter, h = panel_thickness + 1, center = true);
}


jack_height = 10;
toggle_height = 15;
pot_height = 22.5;

buffer = 1.4;

jack2toggle_y = (jack_height + toggle_height) / 2 + buffer;
jack2jack_y = jack_height + buffer;
jack2pot_y = (jack_height + pot_height) / 2 + buffer;

jack1_y = 11;
toggle1_y = jack1_y + jack2toggle_y;
jack2_y = toggle1_y + jack2toggle_y;
jack3_y = jack2_y + jack2toggle_y;
toggle2_y = jack3_y + jack2toggle_y;
jack4_y = toggle2_y + jack2toggle_y;
pot1_y = jack4_y + jack2pot_y;
jack5_y = pot1_y + jack2pot_y;

module panel() {
    difference() {
        // Base panel
        cube([panel_width, panel_height, panel_thickness]);

        // Mounting holes
        mounting_hole(panel_width - mount_offset_x, panel_height - mount_offset_y);
        mounting_hole(mount_offset_x, mount_offset_y);

        // Components (Jacks, Toggles, Pot)
        jack(jack1_y);
        toggle(toggle1_y);
        jack(jack2_y);
        jack(jack3_y);
        toggle(toggle2_y);
        jack(jack4_y);
        pot(pot1_y);
        jack(jack5_y);
    }
}

panel();
