package lepton

import (
	"testing"
)

const (
	hw  = "hw:(contents:(host:examples/hw))"
	lib = "lib:(children:(x86_64-linux-gnu:(children:(libc.so.6:(contents:(host:/lib/x86_64-linux-gnu/libc.so.6))ld-2.23.so:(contents:(host:/lib/x86_64-linux-gnu/id-2.23.so))))))"
)

func TestContentNodeSerialization(t *testing.T) {
	root := Node{name: "hw", path: "examples/hw"}
	s := root.String()
	if s != hw {
		t.Errorf("Expected:%v Actual:%v", hw, s)
	}
}

func TestNodeSerialization(t *testing.T) {
	root := Node{name: "lib"}
	root.children = append(root.children, &Node{name: "x86_64-linux-gnu"})
	root.children[0].children = append(root.children[0].children, &Node{name: "libc.so.6", path: "/lib/x86_64-linux-gnu/libc.so.6"})
	root.children[0].children = append(root.children[0].children, &Node{name: "ld-2.23.so", path: "/lib/x86_64-linux-gnu/id-2.23.so"})
	s := root.String()
	if s != lib {
		t.Errorf("Expected:%v Actual:%v", hw, s)
	}
}

func TestManifestSerialization(t *testing.T) {
	m := NewManifest()
	m.AddDebugFlag("fault", 't')
	m.AddEnvironmentVariable("USER", "bobby")
	m.AddEnvironmentVariable("PWD", "/")
	m.SetProgramPath("/hw")
	m.AddKernel("stage3/stage3")
	m.AddLib("/lib/x86_64-linux-gnu/libc.so.6", true)
	m.AddLib("examples/hw", false)
	s := m.String()
	if s != lib {
		t.Errorf("Expected:%v Actual:%v", hw, s)
	}
}

/*
Keep it for testing
(
    #64 bit elf to boot from host
    children:(kernel:(contents:(host:stage3/stage3))
              #user program
              hw:(contents:(host:examples/hw))
              lib:(children:(x86_64-linux-gnu:(children:(
                                                        libc.so.6:(contents:(host:/lib/x86_64-linux-gnu/libc.so.6))
														)
														(
                                                        ld-2.23.so:(contents:(host:/lib/x86_64-linux-gnu/id-2.23.so))
                                                        )

                                                    )
                                      )
                            )
              lib64:(children:(
                                ld-linux-x86-64.so.2:(contents:(host:/lib64/ld-linux-x86-64.so.2))
                             )
                    )
             )

    # filesystem path to elf for kernel to run
    program:/hw
    fault:t
    arguments:[webg poppy]
    environment:(USER:bobby PWD:/)
)
*/
