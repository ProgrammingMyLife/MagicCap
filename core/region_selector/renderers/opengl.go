package renderers

import (
	"errors"
	"github.com/MagicCap/glhf"
	"github.com/getsentry/sentry-go"
	"github.com/go-gl/gl/v3.3-core/gl"
	"github.com/go-gl/glfw/v3.3/glfw"
	"github.com/magiccap/MagicCap/core/mainthread"
	"image"
	"runtime"
)

// This is used to define a RGBA image.
type rgbaImage struct {
	data []byte
	w, h int
}

// This is used to define the OpenGL renderer.
type openGLRenderer struct {
	glfwMonitors   []*glfw.Monitor
	displays       []image.Rectangle
	mousePressCb   func(index int, pos image.Rectangle)
	mouseReleaseCb func(index int, pos image.Rectangle)
	windows        []*glfw.Window
	keyCb          func(Release bool, Index, Key int)
	darkerTextures []*rgbaImage
	normalTextures []*glhf.Texture
	shaders        []*glhf.Shader
}

// ShouldClose is used to say windows should close.
func (r *openGLRenderer) ShouldClose() {
	r.windows[0].SetShouldClose(true)
}

// WindowShouldClose is used to check if a window should close.
func (r *openGLRenderer) WindowShouldClose(index int) bool {
	return r.windows[index].ShouldClose()
}

// DestroyAll is used to destroy all of the windows.
func (r *openGLRenderer) DestroyAll() {
	mainthread.ExecMainThread(func() {
		for _, v := range r.windows {
			v.MakeContextCurrent()
			v.Destroy()
		}
	})
}

// SetKeyCallback is used to handle key callbacks.
func (r *openGLRenderer) SetKeyCallback(Function func(Release bool, index, key int)) {
	r.keyCb = Function
}

// SetMousePressCallback is used to set a mouse callback for when it is pressed.
func (r *openGLRenderer) SetMousePressCallback(Function func(index int, pos image.Rectangle)) {
	r.mousePressCb = Function
}

// SetMouseReleaseCallback is used to set a mouse callback for when it is released.
func (r *openGLRenderer) SetMouseReleaseCallback(Function func(index int, pos image.Rectangle)) {
	r.mouseReleaseCb = Function
}

// PollEvents is used to poll for events.
func (r *openGLRenderer) PollEvents() {
	mainthread.ExecMainThread(glfw.PollEvents)
}

// Init is used to initialise the renderer.
func (r *openGLRenderer) Init(Displays []image.Rectangle, DarkerScreenshots, Screenshots []*image.RGBA) {
	// Set displays.
	r.displays = Displays

	// Remap the monitors to the order of the "displays" array.
	var GLFWMonitorsUnordered []*glfw.Monitor
	mainthread.ExecMainThread(func() {
		GLFWMonitorsUnordered = glfw.GetMonitors()
	})
	r.glfwMonitors = make([]*glfw.Monitor, len(GLFWMonitorsUnordered))
	for _, Monitor := range GLFWMonitorsUnordered {
		x, y := Monitor.GetPos()
		Matches := false
		for i, v := range Displays {
			if v.Bounds().Min.X == x && v.Bounds().Min.Y == y {
				// This is the correct display.
				r.glfwMonitors[i] = Monitor
				Matches = true
				break
			}
		}
		if !Matches {
			panic(errors.New("cannot find matching glfw display"))
		}
	}

	// Defines all needed OpenGL definitions.
	r.shaders = make([]*glhf.Shader, len(r.displays))
	r.darkerTextures = make([]*rgbaImage, len(r.displays))
	r.normalTextures = make([]*glhf.Texture, len(r.displays))

	// Make a window on each display.
	r.windows = make([]*glfw.Window, len(r.glfwMonitors))
	var FirstWindow *glfw.Window
	mainthread.ExecMainThread(func() {
		for i, v := range r.displays {
			// Creates the window.
			var Window *glfw.Window
			var err error

			// Creates the OpenGL context.
			glfw.WindowHint(glfw.ContextVersionMajor, 3)
			glfw.WindowHint(glfw.ContextVersionMinor, 3)
			glfw.WindowHint(glfw.OpenGLProfile, glfw.OpenGLCoreProfile)
			glfw.WindowHint(glfw.OpenGLForwardCompatible, glfw.True)

			// Sets all of the used window hints.
			glfw.WindowHint(glfw.CenterCursor, glfw.False)
			glfw.WindowHint(glfw.Decorated, glfw.False)
			glfw.WindowHint(glfw.FocusOnShow, glfw.True)
			glfw.WindowHint(glfw.Floating, glfw.True)
			glfw.WindowHint(glfw.AutoIconify, glfw.False)
			glfw.WindowHint(glfw.Resizable, glfw.False)

			// Create the display window.
			monitor := r.glfwMonitors[i]
			if runtime.GOOS == "linux" {
				// Apparently Linux tries to do shit with decorations if the monitor isn't nil and the window is visible.
				// FFS.
				monitor = nil
			}
			width := v.Max.X-v.Min.X
			height := v.Max.Y-v.Min.Y
			Window, err = glfw.CreateWindow(width, height, "MagicCap Region Selector", monitor, FirstWindow)
			if err != nil {
				sentry.CaptureException(err)
				panic(err)
			}
			if FirstWindow == nil {
				FirstWindow = Window
			}
			r.windows[i] = Window
			Window.MakeContextCurrent()
			if runtime.GOOS == "linux" {
				// Set the monitor on Linux.

				// Get the refresh rate first. This stops the screen going black with some GPU's.
				refreshRate := r.glfwMonitors[i].GetVideoMode().RefreshRate

				// Set the monitor.
				Window.SetMonitor(r.glfwMonitors[i], 0, 0, width, height, refreshRate)
			}

			// Remember these for later.
			index := i
			DisplayPos := v

			// Sets the mouse button handler.
			Window.SetMouseButtonCallback(func(_ *glfw.Window, button glfw.MouseButton, action glfw.Action, _ glfw.ModifierKey) {
				if button != glfw.MouseButton1 {
					return
				}

				if action == glfw.Press {
					if r.mousePressCb != nil {
						r.mousePressCb(index, DisplayPos)
					}
				} else if action == glfw.Release {
					if r.mouseReleaseCb != nil {
						r.mouseReleaseCb(index, DisplayPos)
					}
				}
			})

			// Sets the key handler.
			Window.SetKeyCallback(func(_ *glfw.Window, key glfw.Key, _ int, action glfw.Action, _ glfw.ModifierKey) {
				if r.keyCb != nil {
					r.keyCb(action == glfw.Release, index, int(key))
				}
			})

			// Creates all required OpenGL definitions.
			s, err := glhf.NewShader(glhf.AttrFormat{
				{Name: "position", Type: glhf.Vec2},
				{Name: "texture", Type: glhf.Vec2},
			}, glhf.AttrFormat{}, vertexShader, fragmentShader)
			if err != nil {
				panic(err)
			}
			r.shaders[i] = s

			// Creates the texture.
			r.darkerTextures[i] = &rgbaImage{
				data: DarkerScreenshots[i].Pix,
				w:    DarkerScreenshots[i].Bounds().Dx(),
				h:    DarkerScreenshots[i].Bounds().Dy(),
			}

			// Creates the brighter texture.
			t := glhf.NewTexture(
				Screenshots[i].Bounds().Dx(),
				Screenshots[i].Bounds().Dy(),
				true,
				Screenshots[i].Pix,
			)
			r.normalTextures[i] = t
		}
	})
}

type openGlTexture struct {
	texture *glhf.Texture
}

// Begin defines the start of texture modifications.
func (t *openGlTexture) Begin() {
	mainthread.ExecMainThread(t.texture.Begin)
}

// End defines the end of texture modifications.
func (t *openGlTexture) End() {
	mainthread.ExecMainThread(t.texture.End)
}

// SetPixels is used to set the pixels.
func (t *openGlTexture) SetPixels(X, Y, Width, Height int, Pix []byte) {
	mainthread.ExecMainThread(func() {
		t.texture.SetPixels(X, Y, Width, Height, Pix)
	})
}

// GetWidthHeight is used to get the width/height.
func (t *openGlTexture) GetWidthHeight() (int, int) {
	var w int
	var h int
	mainthread.ExecMainThread(func() {
		w = t.texture.Width()
		h = t.texture.Height()
	})
	return w, h
}

// GetDarkerTexture is used to get the darker texture.
func (r *openGLRenderer) GetDarkerTexture(index int) Texture {
	var x *glhf.Texture
	mainthread.ExecMainThread(func() {
		t := r.darkerTextures[index]
		x = glhf.NewTexture(t.w, t.h, true, t.data)
		runtime.GC()
	})
	return &openGlTexture{texture: x}
}

// GetNormalTexturePixels is used to get the normal texture pixels.
func (r *openGLRenderer) GetNormalTexturePixels(index, Left, Top, W, H int) []uint8 {
	var x []uint8
	mainthread.ExecMainThread(func() {
		t := r.normalTextures[index]
		t.Begin()
		x = t.Pixels(Left, Top, W, H)
		t.End()
		runtime.GC()
	})
	return x
}

// RenderTexture is used to render a texture to the screen.
func (r *openGLRenderer) RenderTexture(index int, t Texture) {
	glt := t.(*openGlTexture).texture
	mainthread.ExecMainThread(func() {
		// Get the window.
		window := r.windows[index]

		// Set the focus of the window.
		window.MakeContextCurrent()

		// Create the vertex slice.
		slice := glhf.MakeVertexSlice(r.shaders[index], 6, 6)
		slice.Begin()
		slice.SetVertexData([]float32{
			-1, -1, 0, 1,
			+1, -1, 1, 1,
			+1, +1, 1, 0,

			-1, -1, 0, 1,
			+1, +1, 1, 0,
			-1, +1, 0, 0,
		})
		slice.End()

		// Clear the window.
		glhf.Clear(1, 1, 1, 1)

		// Get the shader.
		shader := r.shaders[index]

		// Render everything.
		shader.Begin()
		glt.Begin()
		slice.Begin()
		slice.Draw()
		slice.End()
		shader.End()
		glt.End()

		// Swap the buffer.
		window.SwapBuffers()

		// Run GC.
		runtime.GC()
	})
}

// RendererInit is used to initialise the renderer.
func (openGLRenderer) RendererInit() {
	err := glfw.Init()
	if err != nil {
		panic(err)
	}
	err = gl.Init()
	if err != nil {
		panic(err)
	}
}