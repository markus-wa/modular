$fn=16;

// Dimensions
panel_width = floor(3 * 5.08); // 3HP
panel_height = 128.5;   // 3U
panel_thickness = 2.00;

// Mounting holes
mount_offset_y = 3;
mount_offset_x = 7.45;
mounting_hole_diameter = 3 + 0.5; // M3

// Compoonent hole sizes
jack_hole_diameter = 6; // 3.5mm
toggle_hole_diameter = 8; // 3.5mm
vga_conn_height = 16.33 + 1.5;
vga_conn_width = 7.9 + 2.0;
vga_height = 30.8;
vga_screws_distance = 25;
vga_screwhole_diameter = 2.5 + 0.5;

jack_height = 10;
toggle_height = 8;

module mounting_hole(x_pos, y_pos) {
    translate([x_pos, y_pos, panel_thickness / 2])
        cylinder(d = mounting_hole_diameter, h = panel_thickness + 1, center = true);
}

module jack(y_pos) {
    translate([panel_width/2, y_pos, panel_thickness / 2])
        cylinder(d = jack_hole_diameter, h = panel_thickness + 1, center = true);
}

module toggle(y_pos) {
    translate([panel_width/2, y_pos, panel_thickness / 2])
        cylinder(d = toggle_hole_diameter, h = panel_thickness + 1, center = true);
}

module vga(y_pos) {
    translate([(panel_width - vga_conn_width)/2, y_pos + ((vga_height-vga_conn_height) / 2) , -1])
        cube([vga_conn_width, vga_conn_height, panel_thickness + 2]);

    translate([panel_width/2, y_pos + (vga_height/2) - (vga_screws_distance/2), panel_thickness / 2])
        cylinder(d = vga_screwhole_diameter, h = panel_thickness + 1, center = true);

    translate([panel_width/2, y_pos + (vga_height/2) + (vga_screws_distance/2), panel_thickness / 2])
        cylinder(d = vga_screwhole_diameter, h = panel_thickness + 1, center = true);
}


buffer = 0.5;

jack2jack_y = jack_height + 2;

vga_y = 12;
jack1_y = vga_y + vga_height + jack_height/2 + 3;
jack2_y = jack1_y + jack2jack_y;
jack3_y = jack2_y + jack2jack_y;
jack4_y = jack3_y + jack2jack_y;
jack5_y = jack4_y + jack2jack_y;
toggle1_y = jack5_y + (jack_height + toggle_height)/2 + 2;

module panel() {
    difference() {
        // Base panel
        cube([panel_width, panel_height, panel_thickness]);

        // Mounting holes
        mounting_hole(panel_width - mount_offset_x, panel_height - mount_offset_y);
        mounting_hole(mount_offset_x, mount_offset_y);

        // Components (Jacks, Toggles, Pot)
        vga(vga_y);
        jack(jack1_y);
        jack(jack2_y);
        jack(jack3_y);
        jack(jack4_y);
        jack(jack5_y);
        toggle(toggle1_y);
    }
}

panel();
