package main

import (
	"reflect"
	"testing"
)

func TestRewritePacmanArgs(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want []string
	}{
		{
			name: "install",
			args: []string{"-S", "vim"},
			want: []string{"install", "vim"},
		},
		{
			name: "update",
			args: []string{"-Syu"},
			want: []string{"update"},
		},
		{
			name: "update refresh",
			args: []string{"-Syyu"},
			want: []string{"update", "--refresh"},
		},
		{
			name: "update split flags",
			args: []string{"-S", "-y", "-u"},
			want: []string{"update"},
		},
		{
			name: "sync",
			args: []string{"-Sy"},
			want: []string{"sync"},
		},
		{
			name: "sync refresh",
			args: []string{"-Syy"},
			want: []string{"sync", "--refresh"},
		},
		{
			name: "search",
			args: []string{"-Ss", "vim"},
			want: []string{"search", "vim"},
		},
		{
			name: "info sync",
			args: []string{"-Si", "vim"},
			want: []string{"info", "vim"},
		},
		{
			name: "info",
			args: []string{"-Qi", "vim"},
			want: []string{"info", "vim"},
		},
		{
			name: "files",
			args: []string{"-Ql", "vim"},
			want: []string{"files", "vim"},
		},
		{
			name: "search installed",
			args: []string{"-Qs", "vim"},
			want: []string{"search", "--installed", "vim"},
		},
		{
			name: "group list",
			args: []string{"-Sg"},
			want: []string{"group", "list"},
		},
		{
			name: "group info",
			args: []string{"-Sg", "base"},
			want: []string{"group", "info", "base"},
		},
		{
			name: "remove with flags",
			args: []string{"-Rns", "vim"},
			want: []string{"remove", "--nosave", "--recursive", "vim"},
		},
		{
			name: "remove with cascade",
			args: []string{"-Rc", "vim"},
			want: []string{"remove", "--cascade", "vim"},
		},
		{
			name: "pass through non-pacman args",
			args: []string{"install", "vim"},
			want: []string{"install", "vim"},
		},
		{
			name: "preserve flags after command",
			args: []string{"-S", "--needed", "vim"},
			want: []string{"install", "--needed", "vim"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := rewritePacmanArgs(tt.args)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("rewritePacmanArgs(%v) = %v, want %v", tt.args, got, tt.want)
			}
		})
	}
}
